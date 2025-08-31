package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/feeds"
)

type Post struct {
	Caption   string
	MediaURL  string
	Permalink string
}

func main() {
	username := os.Getenv("INSTAGRAM_USERNAME")
	if username == "" {
		username = "hopenest_official"
	}

	posts, err := fetchPosts(username, 8)
	if err != nil {
		log.Fatal(err)
	}

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s Instagram Feed", username),
		Link:        &feeds.Link{Href: fmt.Sprintf("https://www.instagram.com/%s/", username)},
		Description: "Latest posts (scraped)",
	}

	for _, p := range posts {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       p.Caption,
			Link:        &feeds.Link{Href: p.Permalink},
			Description: fmt.Sprintf("<img src='%s' width='300'><br/>%s", p.MediaURL, p.Caption),
		})
	}

	f, err := os.Create("feed.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := feed.WriteRss(f); err != nil {
		log.Fatal(err)
	}

	fmt.Println("feed.xml generated (scraped version)")
}

func fetchPosts(username string, limit int) ([]Post, error) {
	url := fmt.Sprintf("https://www.instagram.com/%s/", username)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// strings.Builder 대신 io.ReadAll 사용
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// JSON 데이터 추출
	re := regexp.MustCompile(`window\._sharedData = (.*);</script>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find sharedData")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, err
	}

	user := data["entry_data"].(map[string]interface{})["ProfilePage"].([]interface{})[0].(map[string]interface{})
	graphql := user["graphql"].(map[string]interface{})
	edgeOwner := graphql["user"].(map[string]interface{})["edge_owner_to_timeline_media"].(map[string]interface{})
	edges := edgeOwner["edges"].([]interface{})

	posts := []Post{}
	for i, edge := range edges {
		if i >= limit {
			break
		}
		node := edge.(map[string]interface{})["node"].(map[string]interface{})
		caption := ""
		if edgesCap, ok := node["edge_media_to_caption"].(map[string]interface{}); ok {
			if arr, ok := edgesCap["edges"].([]interface{}); ok && len(arr) > 0 {
				caption = arr[0].(map[string]interface{})["node"].(map[string]interface{})["text"].(string)
			}
		}
		posts = append(posts, Post{
			Caption:   caption,
			MediaURL:  node["display_url"].(string),
			Permalink: fmt.Sprintf("https://www.instagram.com/p/%s/", node["shortcode"].(string)),
		})
	}

	return posts, nil
}
