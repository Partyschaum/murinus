package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/ChimeraCoder/anaconda"
)

func setupTwitterApi() *anaconda.TwitterApi {
	CONSUMER_KEY := os.Getenv("CONSUMER_KEY")
	CONSUMER_SECRET := os.Getenv("CONSUMER_SECRET")

	ACCESS_TOKEN := os.Getenv("ACCESS_TOKEN")
	ACCESS_TOKEN_SECRET := os.Getenv("ACCESS_TOKEN_SECRET")

	anaconda.SetConsumerKey(CONSUMER_KEY)
	anaconda.SetConsumerSecret(CONSUMER_SECRET)

	return anaconda.NewTwitterApi(ACCESS_TOKEN, ACCESS_TOKEN_SECRET)
}

func main() {
	// setup catching of SIGINT and SIGTERM signals
	var done bool
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("Signal catched... exiting...")
		done = true
	}()

	// open file for saving the timeline
	f, err := os.Create("timeline.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	api := setupTwitterApi()

	const fetch int = 10

	// fetch tweets in bunches of 10
	v := url.Values{}
	v.Add("count", strconv.Itoa(fetch))

	var fetchedTweets int

	for {
		if done {
			log.Printf("Exiting... Fetched %d tweets!\n", fetchedTweets)
			w.Flush()
			os.Exit(0)
		}
		timeline, err := api.GetUserTimeline(v)
		if err != nil {
			log.Printf("Error while fetching tweets from timeline: %s\n", err)
		}

		var lastTweetId int64
		for _, tweet := range timeline {
			fmt.Fprintf(w, "Id: %d Created: %s Text: %s\n",
				tweet.Id,
				tweet.CreatedAt,
				tweet.Text,
			)
			lastTweetId = tweet.Id
		}

		v.Set("max_id", strconv.FormatInt(lastTweetId-1, 10))
		fetchedTweets += fetch
		if fetchedTweets%50 == 0 {
			log.Printf("Fetched %d tweets...\n", fetchedTweets)
		}
	}
}
