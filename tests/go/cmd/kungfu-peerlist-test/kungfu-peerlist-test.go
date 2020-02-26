package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/lsds/KungFu/srcs/go/plan"
)

var (
	peerlistPath = flag.String("path", "./hostlists/hostlist.json", "")
	period       = flag.Duration("period", 30*time.Second, "")
)

func main() {
	flag.Parse()
	var peerList plan.PeerList

	content, err := ioutil.ReadFile(*peerlistPath)
	if err != nil {
		fmt.Println("Cannot read file")
	}
	err = json.Unmarshal(content, &peerList)
	if err != nil {
		fmt.Println("Cannot unmarshal content")
	}
	for i := 0;; i++ {
		newPeerList := peerList

		// the first entry stays as the first entry
		rand.Shuffle(len(newPeerList)-1, func(i, j int) {
			newPeerList[i+1], newPeerList[j+1] = newPeerList[j+1], newPeerList[i+1]
		})

		newNumberOfPeers := rand.Intn(len(newPeerList)-1) + 1
		newPeerList = newPeerList[0:newNumberOfPeers]

		reqBody, err := json.Marshal(newPeerList)
		if err != nil {
			fmt.Println("Cannot marshal peer list")
		}

		_, err = http.Post("http://127.0.0.1:9100/put", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Println("Cannot post request ", err)
		}
		time.Sleep(*period)
	}
}
