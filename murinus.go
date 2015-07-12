package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

func parseArguments() {
	flag.Int("user-id", 0, "The ID of the user for whom to return results for.")
	flag.String("screen-name", "", "The screen name of the user for whom to return results for.")
	flag.Int("since-id", 0, "Returns results with an ID greater than (that is, more recent than) the specified ID.")
	flag.Int("max-id", 0, "Returns results with an ID less than (that is, older than) or equal to the specified ID.")
	flag.Bool("trim-user", true, "When set to true each tweet returned in a timeline will include a user object including only the status authors numerical ID.")
	flag.Bool("include-rts", true, "When set to false, the timeline will strip any native retweets. Note: If you’re using the trim parameter in conjunction with includerts, the retweets will still contain a full user object.")

	flag.Int("count", defaultCount, "Specifies the number of tweets to try and retrieve, up to a maximum of 200 per distinct request.")

	flag.Parse()
}

func getQueryParams() url.Values {
	v := url.Values{}

	// Apply defaults
	v.Set("count", strconv.Itoa(defaultCount))
	v.Set("trim_user", "true")

	// Apply set flags
	flag.Visit(func(f *flag.Flag) {
		param := strings.Replace(f.Name, "-", "_", -1)
		v.Set(param, fmt.Sprintf("%v", f.Value))
	})

	return v
}

const defaultCount int = 10

func main() {
	parseArguments()

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

	queryParams := getQueryParams()
	fmt.Println(queryParams)

	for {
		if done {
			fmt.Printf("\nFetched %d tweets!\n", fetchedTweets)
			flushWriter(w)
			os.Exit(0)
		}

		timeline, err := api.GetUserTimeline(queryParams)
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
		queryParams.Set("max_id", strconv.FormatInt(lastTweetId-1, 10))

		fetchedTweets += len(timeline)
		if len(timeline) == 0 {
			fmt.Printf("\nNo more tweets left... Fetched %d tweets!\n", fetchedTweets)
			break
		}

		fmt.Print(".")
	}
}
