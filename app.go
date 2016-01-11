package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"sync"

	. "github.com/Financial-Times/golang-app-template/logging"
	"github.com/satori/go.uuid"
)

var uuidsFileName = flag.String("uuids", "", "Path to uuids file")
var readEndpoint = flag.String("read", "", "Native store GET endpoint")
var postEndpoint = flag.String("post", "", "CMS Notifier POST endpoint")
var concurrent = flag.Bool("concurrency", true, "Disable concurrent publish")

var collectionToSystemOrigin = map[string]string{"methode": "methode-web-pub", "wordpress": "wordpress"}
var collections = []string{"methode", "wordpress"}

func main() {
	InitLogs(os.Stdout, os.Stdout, os.Stderr)
	flag.Parse()

	Info.Printf("Concurrency set: [%t].", *concurrent)
	uuids, err := parseUUIDs(*uuidsFileName)
	if err != nil {
		Warn.Printf("Cannot read uuids file: [%v]", err)
		return
	}
	//Info.Printf("Nr of uuids before validation: [%d].", len(uuids))
	for index, id := range uuids {
		if _, err = uuid.FromString(id); err != nil {
			Warn.Printf("Skipping UUID: [%s]. Error: [%v]", id, err)
			uuids = append(uuids[:index], uuids[index+1:]...) // you can't really do this in java, haha
		}
	}
	//Info.Printf("Nr of uuids after validation: [%d].", len(uuids))

	var wg sync.WaitGroup
	wg.Add(len(uuids))
	for _, id := range uuids {
		if *concurrent {
			go republish(id, &wg)
		} else {
			republish(id, &wg)
		}
	}
	wg.Wait()
}

func republish(id string, wg *sync.WaitGroup) {
	defer wg.Done()
	found := false
	for coll, systemOrigin := range collectionToSystemOrigin {
		readURL := *readEndpoint + "/" + coll + "/" + id
		resp, err := http.Get(readURL)
		if err != nil {
			Warn.Printf("GET request failure for UUID [%s]: [%v].", id, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			if resp.StatusCode != 404 {
				Warn.Printf("GET request unsuccessful. Unexpected status [%s] UUID [%s].", resp.Status, id)
			}
			continue
		}
		found = true

		//post content
		client := &http.Client{}
		req, err := http.NewRequest("POST", *postEndpoint, resp.Body)
		if err != nil {
			Warn.Printf("Failure in creating POST request: [%v]", err)
		}
		req.Header.Add("X-Origin-System-Id", systemOrigin)
		resp, err = client.Do(req)
		if err != nil {
			Warn.Printf("POST request failure for UUID [%s]: [%v]", id, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			Warn.Printf("POST request unsuccessful. Unexpected status [%s] UUID [%s].", resp.Status, id)
		} else {
			Info.Printf("Content with UUID [%s] republished successfully.", id)
		}
		return
	}
	if !found {
		Warn.Printf("Content with UUID [%s] could not be found in collections [%v]", id, collections)
	}
}
func parseUUIDs(uuidsFileName string) ([]string, error) {
	// check for JSON Array first
	if ok, uuids := parseUUIDsFromJSONArray(uuidsFileName); ok {
		return uuids, nil
	}

	//fallback to a text file with each UUID on a new line
	file, err := os.Open(uuidsFileName)
	check(err)
	defer file.Close()

	var uuids []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		uuid := scanner.Text()
		//Info.Println(uuid) // Println will add back the final '\n'
		uuids = append(uuids, uuid)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return uuids, nil
}

func parseUUIDsFromJSONArray(uuidsFileName string) (ok bool, uuids []string) {
	file, err := os.Open(uuidsFileName)
	check(err)
	defer file.Close()

	if err = json.NewDecoder(file).Decode(&uuids); err == nil {
		return true, uuids
	}
	return false, nil
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}
