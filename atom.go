package feeds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Generates Atom feed as XML

const ns = "http://www.w3.org/2005/Atom"

type AtomPerson struct {
	Name  string `xml:"name,omitempty"`
	Uri   string `xml:"uri,omitempty"`
	Email string `xml:"email,omitempty"`
}

type AtomSummary struct {
	XMLName xml.Name `xml:"summary"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomContent struct {
	XMLName xml.Name `xml:"content"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomAuthor struct {
	XMLName xml.Name `xml:"author"`
	AtomPerson
}

type AtomContributor struct {
	XMLName xml.Name `xml:"contributor"`
	AtomPerson
}

type AtomEntry struct {
	XMLName     xml.Name `xml:"entry"`
	Title       string   `xml:"title"`   // required
	Updated     string   `xml:"updated"` // required
	Id          string   `xml:"id"`      // required
	Category    string   `xml:"category,omitempty"`
	Content     *AtomContent
	Rights      string `xml:"rights,omitempty"`
	Source      string `xml:"source,omitempty"`
	Published   string `xml:"published,omitempty"`
	Contributor *AtomContributor
	Links       []*AtomLink  `xml:"link"` // required if no child 'content' elements
	Summary     *AtomSummary // required if content has src or content is base64
	Author      *AtomAuthor  // required if feed lacks an author
}

type AtomLink struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
}

type AtomFeed struct {
	XMLName     xml.Name         `xml:"feed"`
	Xmlns       string           `xml:"xmlns,attr"`
	Title       string           `xml:"title"`   // required
	Id          string           `xml:"id"`      // required
	Updated     string           `xml:"updated"` // required
	Category    string           `xml:"category,omitempty"`
	Icon        string           `xml:"icon,omitempty"`
	Logo        string           `xml:"logo,omitempty"`
	Rights      string           `xml:"rights,omitempty"` // copyright used
	Subtitle    string           `xml:"subtitle,omitempty"`
	Links       []*AtomLink      `xml:"link"`
	Author      *AtomAuthor      `xml:"author"` // required
	Contributor *AtomContributor `xml:"contributor"`
	Entries     []*AtomEntry     `xml:"entry"`
}

type Atom struct {
	*Feed
}

func newAtomEntry(i *Item) *AtomEntry {
	id := i.Id
	// assume the description is html
	c := &AtomContent{Content: i.Description, Type: "html"}

	if len(id) == 0 {
		// if there's no id set, try to create one, either from data or just a uuid
		if len(i.Link.Href) > 0 && (!i.Created.IsZero() || !i.Updated.IsZero()) {
			dateStr := anyTimeFormat("2006-01-02", i.Updated, i.Created)
			host, path := i.Link.Href, "/invalid.html"
			if url, err := url.Parse(i.Link.Href); err == nil {
				host, path = url.Host, url.Path
			}
			id = fmt.Sprintf("tag:%s,%s:%s", host, dateStr, path)
		} else {
			id = "urn:uuid:" + NewUUID().String()
		}
	}
	var name, email string
	if i.Author != nil {
		name, email = i.Author.Name, i.Author.Email
	}

	x := &AtomEntry{
		Title:   i.Title,
		Links:   []*AtomLink{&AtomLink{Href: i.Link.Href, Rel: i.Link.Rel}},
		Content: c,
		Id:      id,
		Updated: anyTimeFormat(time.RFC3339, i.Updated, i.Created),
	}
	if len(name) > 0 || len(email) > 0 {
		x.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: name, Email: email}}
	}
	return x
}

// create a new AtomFeed with a generic Feed struct's data
func (a *Atom) AtomFeed() *AtomFeed {
	updated := anyTimeFormat(time.RFC3339, a.Updated, a.Created)
	feed := &AtomFeed{
		Xmlns:    ns,
		Title:    a.Title,
		Links:    []*AtomLink{&AtomLink{Href: a.Link.Href, Rel: a.Link.Rel}},
		Subtitle: a.Description,
		Id:       a.Link.Href,
		Updated:  updated,
		Rights:   a.Copyright,
	}
	if a.Author != nil {
		feed.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: a.Author.Name, Email: a.Author.Email}}
	} else {
		feed.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: "", Email: ""}}
	}
	for _, e := range a.Items {
		feed.Entries = append(feed.Entries, newAtomEntry(e))
	}
	return feed
}

// return an XML-Ready object for an Atom object
func (a *Atom) FeedXml() interface{} {
	return a.AtomFeed()
}

// return an XML-ready object for an AtomFeed object
func (a *AtomFeed) FeedXml() interface{} {
	return a
}

func (a *AtomFeed) Link(rel string) (string, bool) {
	for _, link := range a.Links {
		if link.Rel == rel {
			return link.Href, true
		}
	}

	return "", false
}

func (a *AtomEntry) Link(rel string) (string, bool) {
	for _, link := range a.Links {
		if link.Rel == rel {
			return link.Href, true
		}
	}

	return "", false
}

func ParseAtomFeed(content string) (*AtomFeed, error) {
	var feed AtomFeed
	decoder := xml.NewDecoder(bytes.NewBufferString(content))
	decoder.Strict = true

	if err := decoder.Decode(&feed); err != nil {
		return nil, err
	}

	fmt.Printf("%+s", feed)

	return &feed, nil
}

func DownloadAtomFeed(url string) (*AtomFeed, error) {
	client := &http.Client{}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Accept", "application/atom+xml")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var feed AtomFeed
	decoder := xml.NewDecoder(response.Body)
	if err := decoder.Decode(&feed); err != nil {
		return nil, err
	}

	return &feed, nil
}
