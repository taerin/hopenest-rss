package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/feeds"
)

type Post struct {
	Caption   string
	MediaURL  string
	Permalink string
}

// Instagram JSON 구조
type Graphql struct {
	User struct {
		EdgeOwnerToTimelineMedia struct {
			Edges []struct {
				Node struct {
					Shortcode          string `json:"shortcode"`
					DisplayURL         string `json:"display_url"`
					EdgeMediaToCaption struct {
						Edges []struct {
							Node struct {
								Text string `json:"text"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"edge_media_to_caption"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"edge_owner_to_timeline_media"`
	} `json:"user"`
}

type InstaResponse struct {
	Graphql Graphql `json:"graphql"`
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
	url := fmt.Sprintf("https://www.instagram.com/%s/?__a=1&__d=dis", username)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data InstaResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	posts := []Post{}
	edges := data.Graphql.User.EdgeOwnerToTimelineMedia.Edges
	for i, edge := range edges {
		if i >= limit {
			break
		}
		caption := ""
		if len(edge.Node.EdgeMediaToCaption.Edges) > 0 {
			caption = edge.Node.EdgeMediaToCaption.Edges[0].Node.Text
		}
		posts = append(posts, Post{
			Caption:   caption,
			MediaURL:  edge.Node.DisplayURL,
			Permalink: fmt.Sprintf("https://www.instagram.com/p/%s/", edge.Node.Shortcode),
		})
	}

	return posts, nil
}
