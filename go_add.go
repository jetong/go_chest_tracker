package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	timestamp := time.Now().Unix()

	// parse command line args into variables
	name := os.Args[1]
	days := os.Args[2]
	hours := os.Args[3]
	mins := os.Args[4]
	available_chests := os.Args[5]

	// read .api_key.txt
	key, err := ioutil.ReadFile(".api_key.txt")
	if err != nil {
		panic(err)
	}
	api_key := strings.TrimSuffix(string(key), "\n") // trim newline

	// query riot for summoner id
	api_query_for_id := "https://na1.api.riotgames.com/lol/summoner/v3/summoners/by-name/" + name + "?api_key=" + api_key
	req, _ := http.NewRequest("GET", api_query_for_id, nil)
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}
	id := strconv.Itoa(int(data["id"].(float64)))

	// query for current chest count
	api_query_for_chests := "https://na1.api.riotgames.com/lol/champion-mastery/v3/champion-masteries/by-summoner/" + id + "?api_key=" + api_key
	req, _ = http.NewRequest("GET", api_query_for_chests, nil)
	res, _ = http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ = ioutil.ReadAll(res.Body)
	var dat []interface{}
	if err := json.Unmarshal(body, &dat); err != nil {
		panic(err)
	}
	// count the number of champions whose chestGranted is true
	total_chests := 0
	for _, mapInterface := range dat {
		m := mapInterface.(map[string]interface{})
		if m["chestGranted"].(bool) {
			total_chests++
		}
	}

	// write data to file
	f, err := os.OpenFile("lol_data.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	entry := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v:%v\n", name, id, days, hours, mins, timestamp, total_chests, available_chests)
	if _, err = f.WriteString(entry); err != nil {
		panic(err)
	}
}
