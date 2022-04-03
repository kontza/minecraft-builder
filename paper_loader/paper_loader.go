package paper_loader

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gocolly/colly"
)

type Projects struct {
	Projects []string
}

type Versions struct {
	Versions []string
}

type Builds struct {
	Builds []int
}

type ApplicationDownload struct {
	Name string
}

type BuildDownloads struct {
	Application ApplicationDownload
}

type Build struct {
	Downloads BuildDownloads
}

type ProjectsCallback func(*[]string)

type PaperLoader interface {
	LoadLatest(f ProjectsCallback)
	LoadProject(project string)
}

type paperLoader struct {
	collector *colly.Collector
}

func (l paperLoader) download(project string, version string, build int, artifact string) {
	log.Println("Artifact to load", artifact)
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/builds/277/downloads/paper-1.18.2-277.jar
	log.Println(fmt.Sprintf("https://papermc.io/api/v2/projects/%s/versions/%s/builds/%d/downloads/%s", project, version, build, artifact))
}

func (l paperLoader) loadBuild(project string, version string, buildNumber int) {
	log.Println("Build to load", buildNumber)
	c := l.collector.Clone()
	c.OnResponse(func(r *colly.Response) {
		var build Build
		if err := json.Unmarshal(r.Body, &build); err != nil {
			log.Println("Failed to unmarshal build:", err)
		} else {
			l.download(project, version, buildNumber, build.Downloads.Application.Name)
		}
	})
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/builds/277
	c.Visit(fmt.Sprintf("https://papermc.io/api/v2/projects/%s/versions/%s/builds/%d", project, version, buildNumber))
}

func (l paperLoader) loadVersion(project string, version string) {
	log.Println("Version to load", version)
	c := l.collector.Clone()
	c.OnResponse(func(r *colly.Response) {
		var builds Builds
		if err := json.Unmarshal(r.Body, &builds); err != nil {
			log.Println("Failed to unmarshal builds:", err)
		} else {
			l.loadBuild(project, version, builds.Builds[len(builds.Builds)-1])
		}
	})
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/
	c.Visit(fmt.Sprintf("https://papermc.io/api/v2/projects/%s/versions/%s", project, version))
}

func (l paperLoader) LoadProject(project string) {
	log.Println("Project to load", project)
	c := l.collector.Clone()
	c.OnResponse(func(r *colly.Response) {
		var versions Versions
		if err := json.Unmarshal(r.Body, &versions); err != nil {
			log.Println("Failed to unmarshal versions:", err)
		} else {
			l.loadVersion(project, versions.Versions[len(versions.Versions)-1])
		}
	})
	c.Visit(fmt.Sprintf("https://papermc.io/api/v2/projects/%s", project))
}

func (l paperLoader) LoadLatest(f ProjectsCallback) {
	c := l.collector.Clone()
	c.OnResponse(func(r *colly.Response) {
		var projects Projects
		if err := json.Unmarshal(r.Body, &projects); err != nil {
			log.Println("Failed to unmarshal projects:", err)
		} else {
			log.Println("URL", r.Request.URL)
			f(&projects.Projects)
		}
	})
	c.Visit("https://papermc.io/api/v2/projects")
}

func NewPaperLoader() PaperLoader {
	loader := &paperLoader{
		collector: colly.NewCollector(),
	}
	loader.collector.OnError(func(r *colly.Response, err error) {
		log.Println("Failed to find the latest:", err)
	})
	loader.collector.OnScraped(func(r *colly.Response) {
		log.Println("Scrape done.", r.Request.URL)
	})
	return loader
}
