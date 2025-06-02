package node

import (
	"log"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/net/html"
)

func GetVersions() []string {
	resp, err := http.Get("https://nodejs.org/dist/")
	if err != nil {
		log.Fatalf("Error fetching versions: %v", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing HTML: %v", err)
	}

	var versions []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" &&
					strings.HasPrefix(attr.Val, "v") &&
					strings.HasSuffix(attr.Val, "/") &&
					semver.IsValid(attr.Val[:len(attr.Val)-1]) {
					version := strings.TrimSuffix(attr.Val, "/")
					versions = append(versions, version)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	return versions
}
