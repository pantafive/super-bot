package bot

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/lcw"
	"github.com/pkg/errors"
)

// Anecdote bot, returns from https://jokesrv.rubedo.cloud/
type Anecdote struct {
	client     HTTPClient
	categCache lcw.LoadingCache
}

// NewAnecdote makes a bot for http://rzhunemogu.ru
func NewAnecdote(client HTTPClient) *Anecdote {
	log.Printf("[INFO] anecdote bot with https://jokesrv.rubedo.cloud/ and http://api.icndb.com/jokes/random")
	c, _ := lcw.NewExpirableCache(lcw.MaxKeys(100), lcw.TTL(time.Hour))
	return &Anecdote{client: client, categCache: c}
}

// Help returns help message
func (a Anecdote) Help() string {
	return genHelpMsg(a.ReactOn(), "расскажет анекдот или шутку")
}

// OnMessage returns one entry
func (a Anecdote) OnMessage(msg Message) (response Response) {

	if !contains(a.ReactOn(), msg.Text) {
		return NewVoidResponse()
	}

	if contains([]string{"chuck!", "/chuck"}, msg.Text) {
		return a.chuck()
	}

	cc, err := a.categories()
	if err != nil {
		log.Printf("[WARN] category retrival failed, %v", err)
	}

	switch {
	case contains([]string{"chuck!", "/chuck"}, msg.Text):
		return a.chuck()
	case contains(cc, msg.Text):
		return a.jokesrv(strings.TrimSuffix(strings.TrimPrefix(msg.Text, "/"), "!"))
	default:
		return a.jokesrv("oneliner")
	}

}

// get categorise from https://jokesrv.rubedo.cloud/categories and extend with / prefix and ! suffix
// to mach commands
func (a Anecdote) categories() ([]string, error) {
	res, err := a.categCache.Get("categories", func() (interface{}, error) {
		var categories []string
		req, err := http.NewRequest("GET", "https://jokesrv.rubedo.cloud/categories", nil)
		if err != nil {
			return nil, errors.Wrapf(err, "can't make categories request")
		}
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "can't send categories request")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Errorf("bad response code %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&categories)
		return categories, errors.Wrap(err, "can't decode category response")
	})
	if err != nil {
		return nil, err
	}

	var cc []string
	for _, c := range res.([]string) {
		cc = append(cc, c+"!")
	}
	return cc, nil
}

func (a Anecdote) jokesrv(category string) (response Response) {
	reqURL := "https://jokesrv.rubedo.cloud/" + category

	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return NewVoidResponse()
	}
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return NewVoidResponse()
	}
	defer resp.Body.Close()
	rr := struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		log.Printf("[WARN] failed to parse body, error=%v", err)
		return NewVoidResponse()

	}

	return NewResponse(strings.TrimSuffix(rr.Content, "."), true, false, false, false, 0)
}

func (a Anecdote) chuck() (response Response) {

	chuckResp := struct {
		Type  string
		Value struct {
			Categories []string
			Joke       string
		}
	}{}

	reqURL := "http://api.icndb.com/jokes/random"
	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return NewVoidResponse()
	}
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return NewVoidResponse()
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&chuckResp); err != nil {
		log.Printf("[WARN] failed to convert from json, error=%v", err)
		return NewVoidResponse()
	}
	return NewResponse(
		"- "+strings.Replace(chuckResp.Value.Joke, "&quot;", "\"", -1),
		true,
		false,
		false,
		false,
		0,
	)
}

// ReactOn keys
func (a Anecdote) ReactOn() []string {

	cc, err := a.categories()
	if err != nil {
		log.Printf("[WARN] category retrival failed, %v", err)
	}

	return append([]string{"анекдот!", "анкедот!", "joke!", "chuck!"}, cc...)
}
