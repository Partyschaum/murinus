package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/template"

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

const fetch int = 10

func flushWriter(w *bufio.Writer) {
	if err := w.Flush(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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

	// fetch tweets in bunches of 10
	v := url.Values{}
	v.Add("count", strconv.Itoa(fetch))

	var fetchedTweets int

	// initialize tweet template
	tmpl, err := ioutil.ReadFile("templates/tweet.tmpl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	template := template.Must(template.New("tweet").Parse(string(tmpl)))

	for {
		if done {
			fmt.Printf("Exiting... Fetched %d tweets!\n", fetchedTweets)
			flushWriter(w)
			os.Exit(0)
		}
		timeline, err := api.GetUserTimeline(v)
		if err != nil {
			flushWriter(w)
			fmt.Printf("Error while fetching tweets from timeline: %s\n", err)
			os.Exit(1)
		}

		var lastTweetId int64
		for _, tweet := range timeline {
			template.Execute(w, tweet)
			lastTweetId = tweet.Id
		}
		if len(timeline) > 0 {
			lastTweetId = timeline[len(timeline)-1].Id
		} else {
			fmt.Printf("No more tweets left... Fetched %d tweets!\n", fetchedTweets)
			break
		}

		v.Set("max_id", strconv.FormatInt(lastTweetId-1, 10))
		fetchedTweets += len(timeline)
		if fetchedTweets%50 == 0 {
			fmt.Printf("Fetched %d tweets...\n", fetchedTweets)
		}

		flushWriter(w)
	}
}
