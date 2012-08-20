package main

import (
	"encoding/json"
	"github.com/bsdf/twitter"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	// "strconv"
	"config"
	"fmt"
	"strconv"
	// "time"
	"unicode/utf8"
)

//https://api.weibo.com/oauth2/authorize?client_id=3558864612&redirect_uri=http://cn.myalert.info/test.php&response_type=token
//curl --data "client_id=3558864612&client_secret=b50f7e096d4048ab39f151888e628345&grant_type=authorization_code&code=ce4359430f05c73bf0c11f45ab8d0621&redirect_uri=http://cn.myalert.info/test.php" https://api.weibo.com/oauth2/access_token

//https://api.weibo.com/2/statuses/user_timeline.json?access_token=2.008TkTLDIQdqsD2b947e60590yQenc&screen_name=hugozhu

func SubStringByByte(str string, len2 int) string {
	if len(str) <= len2 {
		return str
	}
	n := 0
	for c := range str {
		if c > len2 {
			return str[:n]
		}
		n = c
	}
	return str
}

func SubStringByChar(str string, len int) string {
	c, n := 0, 0
	for c = range str {
		n++
		if n > len {
			return str[0:c]
		}
	}
	return str
}

type Post struct {
	Id   int64
	Text string
}

//3480730133033379
func Timeline(access_token string, screen_name string, since_id int64) []Post {
	url := "https://api.weibo.com/2/statuses/user_timeline.json?access_token=" + access_token
	url += fmt.Sprintf("&screen_name=%s&since_id=%d", screen_name, since_id)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bytes, _ := ioutil.ReadAll(resp.Body)

	log.Println(url)

	var posts = []Post{}
	if resp.StatusCode == 200 {
		var data map[string]interface{}
		json.Unmarshal(bytes, &data)

		// log.Println(string(bytes))

		if data["statuses"] != nil {
			for _, entry := range data["statuses"].([]interface{}) {
				entry := entry.(map[string]interface{})
				id, _ := strconv.ParseInt(entry["idstr"].(string), 10, 64)
				text := entry["text"].(string)
				link := ""
				if entry["original_pic"] != nil {
					link = " ✈ " + entry["original_pic"].(string)
				}

				if entry["retweeted_status"] != nil {
					retweeted := entry["retweeted_status"].(map[string]interface{})
					if retweeted["user"] != nil {
						re_user := retweeted["user"].(map[string]interface{})
						text = text + " //RT @" + re_user["name"].(string) + ": " + retweeted["text"].(string)
					}

					if retweeted["original_pic"] != nil {
						link = " ✈ " + retweeted["original_pic"].(string)
					}
				}
				len1 := utf8.RuneCount([]byte(text))
				len2 := utf8.RuneCount([]byte(link))
				if len1+len2 > 140 {
					text = SubStringByChar(text, 140-len2) + link
				} else {
					text = text + link
				}

				posts = append(posts, Post{id, text})
			}
		}
	} else {
		log.Fatal(string(bytes))
	}
	return posts
}

const (
	Twitter_ConsumerKey    = "5QrdEQ39A1yZJcMAuc2mwg"
	Twitter_ConsumerSecret = "bJattIehGRzbe67ei6dgSx8KGHYuj4KbI0lqVBQMQ"
)

var ACCESS_TOKEN string

func sync(name string, user *config.User) {
	if user.Enabled {
		weibo_account := user.GetAccount("tsina")
		twitter_account := user.GetAccount("twitter")
		posts := Timeline(ACCESS_TOKEN, weibo_account.Name, user.Last_weibo_id)
		t := twitter.Twitter{
			ConsumerKey:      Twitter_ConsumerKey,
			ConsumerSecret:   Twitter_ConsumerSecret,
			OAuthToken:       twitter_account.Oauth_token_key,
			OAuthTokenSecret: twitter_account.Oauth_token_secret,
		}
		for i := len(posts) - 1; i >= 0; i-- {
			post := posts[i]
			if post.Id > user.Last_weibo_id {
				user.Last_weibo_id = post.Id
				tweet, err := t.Tweet(post.Text)
				log.Println(weibo_account.Name, post.Text)
				if err != nil {
					log.Println("[error]", tweet, err)
				}
			}
		}
	}
}

var complete_chan chan string

func main() {
	conf := config.NewConfig(os.Getenv("PWD") + "/config.json")
	defer func() {
		conf.Save()
	}()

	ACCESS_TOKEN = "2.008TkTLDIQdqsDbb35ec359cwkV8CB"

	n := len(conf.Users())
	complete_chan := make(chan string, n)
	for name, user := range conf.Users() {
		go func(n string, u *config.User) {
			sync(n, u)
			complete_chan <- n + " is done"
		}(name, user)
	}
	log.Println("wait for complete")
	for i := 0; i < n; i++ {
		log.Print(<-complete_chan)
	}
	log.Println("done")
}
