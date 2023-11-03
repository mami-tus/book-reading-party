package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/text/encoding/japanese"
)

var pageURLFormat = "https://www.aozora.gr.jp/cards/%s/card%s.html"
type Entry struct {
	AuthorID string
	Author   string
	TitleID  string
	Title    string
	SiteURL  string
	ZipURL   string
}

func findEntries(siteURL string) ([]Entry, error) {
	// Deprecated:
	// doc, err := goquery.NewDocument(siteURL)
	// まず、http.Get関数を使用してウェブページからHTTP GETリクエストを送信し、そのレスポンスを取得しています。エラーが発生した場合は、空の文字列を返します。
	res, err := http.Get(siteURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// 次に、goquery.NewDocumentFromReader関数を使用して、レスポンスボディからHTMLドキュメントを作成します。エラーが発生した場合も、空の文字列を返します。
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	// リスト9.3
	// doc.Find("ol li a").Each(func(n int, elem *goquery.Selection) {
	// 	fmt.Println(elem.Text(), elem.AttrOr("href", ""))
	// })

	// リスト9.4
	// pat := regexp.MustCompile(`.*/cards/([0-9]+)/card([0-9]+).html$`)
	// doc.Find("ol li a").Each(func(n int, elem *goquery.Selection) {
	// 	token := pat.FindStringSubmatch(elem.AttrOr("href", ""))
	// 	if len(token) != 3 {
	// 		return
	// 	}
	// 	pageURL := fmt.Sprintf("https://www.aozora.gr.jp/cards/%s/card%s.html", token[1], token[2])
	// 	fmt.Println(pageURL)
	// })

	// リスト9.5 - 9.9
	entries := []Entry{}
	pat := regexp.MustCompile(`.*/cards/([0-9]+)/card([0-9]+).html$`)
	doc.Find("ol li a").Each(func(n int, elem *goquery.Selection) {
		token := pat.FindStringSubmatch(elem.AttrOr("href", ""))
		if len(token) != 3 {
			return
		}
		title := elem.Text()
		pageURL := fmt.Sprintf(pageURLFormat, token[1], token[2])
		author, zipURL := findAuthorAndZIP(pageURL)
		if zipURL != "" {
			entries = append(entries, Entry{
				AuthorID: token[1],
				Author:   author,
				TitleID:  token[2],
				Title:    title,
				SiteURL:  siteURL,
				ZipURL:   zipURL,
			})
		}

	})

	return entries, nil
}

// リスト9.6
func findAuthorAndZIP(siteURL string) (string, string) {
	// Deprecated:
	// doc, err := goquery.NewDocument(siteURL)
	res, err := http.Get(siteURL)
	if err != nil {
		return "", ""
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", ""
	}
	// author := doc.Find("table[summary='作家データ'] tr:nth-child(1) td:nth-child(2)").Text()
	author := doc.Find("table[summary='作家データ'] tr:nth-child(2) td:nth-child(2)").First().Text()

	zipURL := ""
	doc.Find("table.download a").Each(func(n int, elem *goquery.Selection) {
		href := elem.AttrOr("href", "")
		if strings.HasSuffix(href, ".zip") {
			zipURL = href
		}
	})

	if zipURL == "" {
		return author, zipURL
	}

	if strings.HasPrefix(zipURL, "http://") || strings.HasPrefix(zipURL, "https://") {
		return author, zipURL
	}
	u, err := url.Parse(siteURL)
	if err != nil {
		return author, ""
	}
	u.Path = path.Join(path.Dir(u.Path), zipURL)
	return author, u.String()
}

func setupDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS authors (
			author_id TEXT, author TEXT, PRIMARY KEY(author_id))`,
		`CREATE TABLE IF NOT EXISTS contents (
			author_id TEXT, title_id TEXT, title TEXT, content TEXT, PRIMARY KEY(author_id, title_id))`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS contents_fts USING fts4(words)`,
	}
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatal(err)
		}
	}
	return db, nil
}

func addEntry(db *sql.DB, entry *Entry, content string) error {
	_, err := db.Exec(
		`REPLACE INTO authors(author_id, author) values(?, ?)`,
		entry.AuthorID,
		entry.Author,
	)
	if err != nil {
		return err
	}

	res, err := db.Exec(
		`REPLACE INTO contents(author_id, title_id, title, content) values(?, ?, ?, ?)`,
		entry.AuthorID,
		entry.TitleID,
		entry.Title,
		content,
	)
	if err != nil {
		return err
	}
	docID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return err
	}
	seg := t.Wakati(content)
	_, err = db.Exec(
		`REPLACE INTO contents_fts(docid, words) values(?, ?)`,
		docID,
		strings.Join(seg, " "),
	)
	if err != nil {
		return err
	}
	return nil

}
func main() {
	db, err := setupDB("database.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	listURL := "https://www.aozora.gr.jp/index_pages/person879.html"

	entries, err := findEntries(listURL)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("found %d entries", len(entries))
	for _, entry := range entries {
		log.Printf("adding %+v\n", entry)
		content, err := extractText(entry.ZipURL)
		if err != nil {
			log.Println(err)
			continue
		}
		err = addEntry(db, &entry, content)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

// リスト9.10
func extractText(zipURL string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}

	for _, file := range r.File {
		if path.Ext(file.Name) == ".txt" {
			f, err := file.Open()
			if err != nil {
				return "", err
			}
			b, err := ioutil.ReadAll(f)
			f.Close()
			if err != nil {
				return "", err
			}
			b, err = japanese.ShiftJIS.NewDecoder().Bytes(b)
			if err != nil {
				return "", err
			}
			return string(b), nil // ここでデコードされた文字列を返します
		}
	}

	return "", errors.New("contents not found")
}
