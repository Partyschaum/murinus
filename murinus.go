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

func flushWriter(w *bufio.Writer) {
	if err := w.Flush(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createFile(file string) *os.File {
	f, err := os.Create(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return f
}

func getTweetTemplate() *template.Template {
	tmpl, err := ioutil.ReadFile("templates/tweet.tmpl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return template.Must(template.New("tweet").Parse(string(tmpl)))
}

func writeTimeline(w *bufio.Writer, timeline []anaconda.Tweet) (int64, error) {
	var lastTweetId int64
	t := getTweetTemplate()
	for _, tweet := range timeline {
		lastTweetId = tweet.Id
		if err := t.Execute(w, tweet); err != nil {
			return lastTweetId, err
		}
	}
	return lastTweetId, nil
}

const fetch int = 10

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

	f := createFile("timeline.json")
	defer f.Close()

	w := bufio.NewWriter(f)

	api := setupTwitterApi()

	var fetchedTweets int

	v := url.Values{}
	v.Add("count", strconv.Itoa(fetch))

	for {
		if done {
			fmt.Printf("\nFetched %d tweets!\n", fetchedTweets)
			flushWriter(w)
			os.Exit(0)
		}

		timeline, err := api.GetUserTimeline(v)
		if err != nil {
			flushWriter(w)
			fmt.Println(err)
			os.Exit(1)
		}

		lastTweetId, err := writeTimeline(w, timeline)
		if err != nil {
			flushWriter(w)
			fmt.Println(err)
			os.Exit(1)
		}
		v.Set("max_id", strconv.FormatInt(lastTweetId-1, 10))

		fetchedTweets += len(timeline)
		if len(timeline) == 0 {
			fmt.Printf("\nNo more tweets left... Fetched %d tweets!\n", fetchedTweets)
			break
		}

		fmt.Print(".")
	}
}
