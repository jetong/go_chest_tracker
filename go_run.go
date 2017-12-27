package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var api_key string

// to track if data has changed in the log
var changed bool = false

// processSummoner() takes a line from lol_data.txt and calls a goroutine to update the available chest count
// returns the updated line as a chan string type
func processSummoner(line string) chan string {
	out := make(chan string)
	go func() {
		changed = false
		fields := strings.Split(line, ":")

		type Summoner struct {
			Name, Id, Days, Hours, Mins, Timestamp, Old_chests, Available_chests string
		}

		s := Summoner{Name: fields[0], Id: fields[1], Days: fields[2], Hours: fields[3], Mins: fields[4], Timestamp: fields[5], Old_chests: fields[6], Available_chests: fields[7]}

		// query summoner data for current chest count
		api_query_for_chests := "https://na1.api.riotgames.com/lol/champion-mastery/v3/champion-masteries/by-summoner/" + s.Id + "?api_key=" + api_key
		req, _ := http.NewRequest("GET", api_query_for_chests, nil)
		res, _ := http.DefaultClient.Do(req)
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		var dat []interface{}
		if err := json.Unmarshal(body, &dat); err != nil {
			panic(err)
		}
		// count the number of champions where chestGranted is true
		current_chests := 0
		for _, mapInterface := range dat {
			m := mapInterface.(map[string]interface{})
			if m["chestGranted"].(bool) {
				current_chests++
			}
		}
		old_chests, _ := strconv.Atoi(s.Old_chests)
		available_chests, _ := strconv.Atoi(s.Available_chests)
		if old_chests < current_chests {
			// a chest has been "consumed" since last checked
			changed = true
			s.Old_chests = strconv.Itoa(current_chests)
			available_chests--
			s.Available_chests = strconv.Itoa(available_chests)
		}

		// if we haven't hit the chest limit, check if a chest has accrued
		const CHEST_LIMIT = 4
		if available_chests < CHEST_LIMIT {
			// convert to integers
			days, _ := strconv.Atoi(s.Days)
			hours, _ := strconv.Atoi(s.Hours)
			mins, _ := strconv.Atoi(s.Mins)
			timestamp, _ := strconv.Atoi(s.Timestamp)

			// convert to seconds
			days = days * 24 * 3600
			hours = hours * 3600
			mins = mins * 60
			next_available_date := timestamp + days + hours + mins

			current_date := time.Now().Unix()
			if int(current_date) > next_available_date {
				// chest has accrued
				changed = true
				s.Timestamp = strconv.Itoa(int(current_date))
				s.Days = "6"
				s.Hours = "23"
				s.Mins = "59"
				available_chests++
				s.Available_chests = strconv.Itoa(available_chests)
			}
		}

		out <- string(s.Name + ":" + s.Id + ":" + s.Days + ":" + s.Hours + ":" + s.Mins + ":" + s.Timestamp + ":" + s.Old_chests + ":" + s.Available_chests)
		close(out)

	}()
	return out
}

func main() {
	// retrieve api key
	key, err := ioutil.ReadFile(".api_key.txt")
	if err != nil {
		panic(err)
	}
	api_key = strings.TrimSuffix(string(key), "\n") // trim newline

	// parse lines from lol_data.txt
	fileContent, err := ioutil.ReadFile("lol_data.txt")
	if err != nil {
		log.Fatalln(err)
	}
	content := strings.TrimSuffix(string(fileContent), "\n") // trim newline
	lines := strings.Split(string(content), "\n")

	// check if data needs update
	// create channels for goroutines
	channels := make([]chan string, len(lines))
	for i := range channels {
		channels[i] = make(chan string)
	}
	// send each line of data to a separate goroutine for processing
	for i, line := range lines {
		channels[i] = processSummoner(line)
	}

	// create header for each log entry
	log_header := "------------- Log Start --------------\n"
	log_header += time.Now().Format(time.RFC822)
	if changed {
		log_header += "   (changed)\n"
	}else{
		log_header += "\n"
	}
	var logWrite string
	logWrite += log_header

	var dataWrite string
	// consolidate processed data
	for _, line := range channels {
		l := fmt.Sprintf("%v\n", <-line)
		dataWrite += l
		logWrite += l
	}

	// write to lol_data.txt and log.txt
	data, err := os.OpenFile("lol_data.txt", os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer data.Close()
	if _, err = data.WriteString(dataWrite); err != nil {
		panic(err)
	}
	log, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer log.Close()
	if _, err = log.WriteString(logWrite); err != nil {
		panic(err)
	}
}
