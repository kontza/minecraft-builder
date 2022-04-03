package paper_loader

import (
	"encoding/json"
	"log"

	"github.com/gocolly/colly"
)

type Projects struct {
	Projects []string
}

type ProjectsCallback func(*[]string)

type PaperLoader interface {
	LoadLatest(f ProjectsCallback)
	LoadProject(project string)
}

type paperLoader struct {
}

func (l paperLoader) LoadProject(project string) {
	log.Println("Should process project", project)
}

func (l paperLoader) LoadLatest(f ProjectsCallback) {
	c := colly.NewCollector()
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Failed to find the latest:", err)
	})
	c.OnScraped(func(r *colly.Response) {
		log.Println("Scrape done.", r.Request.URL)
	})
	c.OnResponse(func(r *colly.Response) {
		var projects Projects
		if err := json.Unmarshal(r.Body, &projects); err != nil {
			log.Println("Failed to unmarshal projects:", err)
		} else {
			// Call BuilderApplication
			f(&projects.Projects)
		}
	})
	c.Visit("https://papermc.io/api/v2/projects")
}

func NewPaperLoader() PaperLoader {
	return &paperLoader{}
}
