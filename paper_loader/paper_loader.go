package paper_loader

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
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
	LoadProject(project string, onProgress func(string))
}

type paperLoader struct {
	url        *url.URL
	collector  *colly.Collector
	onProgress func(string)
}

func (l paperLoader) download(project string, version string, build int, artifact string) {
	client := grab.NewClient()
	req, _ := grab.NewRequest(artifact, fmt.Sprintf("%s/%s/versions/%s/builds/%d/downloads/%s", l.url.String(), project, version, build, artifact))
	req.NoResume = true

	// start download
	l.onProgress(fmt.Sprintln("Starting to download:", artifact))
	resp := client.Do(req)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			l.onProgress(fmt.Sprintf("Downloaded %v / %v bytes\n",
				humanize.Bytes(uint64(resp.BytesComplete())),
				humanize.Bytes(uint64(resp.Size()))))
		case <-resp.Done:
			break Loop
		}
	}

	// check for errors
	if err := resp.Err(); err != nil {
		l.onProgress(fmt.Sprintln("Download failed:", err))
	}
	l.onProgress(fmt.Sprintln("Artifact saved:", resp.Filename))
}

func (l paperLoader) loadBuild(project string, version string, buildNumber int) {
	l.onProgress(fmt.Sprintln("Build to load:", buildNumber))
	c := l.collector.Clone()
	c.OnError(l.logError)
	c.OnResponse(func(r *colly.Response) {
		var build Build
		if err := json.Unmarshal(r.Body, &build); err != nil {
			l.onProgress(fmt.Sprintln("Failed to unmarshal build:", err))
		} else {
			l.download(project, version, buildNumber, build.Downloads.Application.Name)
		}
	})
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/builds/277
	c.Visit(fmt.Sprintf("%s/%s/versions/%s/builds/%d", l.url.String(), project, version, buildNumber))
}

func (l paperLoader) loadVersion(project string, version string) {
	l.onProgress(fmt.Sprintln("Version to load:", version))
	c := l.collector.Clone()
	c.OnError(l.logError)
	c.OnResponse(func(r *colly.Response) {
		var builds Builds
		if err := json.Unmarshal(r.Body, &builds); err != nil {
			l.onProgress(fmt.Sprintln("Failed to unmarshal builds:", err))
		} else {
			l.loadBuild(project, version, builds.Builds[len(builds.Builds)-1])
		}
	})
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/
	c.Visit(fmt.Sprintf("%s/%s/versions/%s", l.url.String(), project, version))
}

func (l paperLoader) LoadProject(project string, onProgress func(string)) {
	l.onProgress = onProgress
	l.onProgress(fmt.Sprintln("Project to load:", project))
	c := l.collector.Clone()
	c.OnError(l.logError)
	c.OnResponse(func(r *colly.Response) {
		var versions Versions
		if err := json.Unmarshal(r.Body, &versions); err != nil {
			l.onProgress(fmt.Sprintln("Failed to unmarshal versions:", err))
		} else {
			l.loadVersion(project, versions.Versions[len(versions.Versions)-1])
		}
	})
	c.Visit(fmt.Sprintf("%s/%s", l.url.String(), project))
}

func (l paperLoader) logError(r *colly.Response, err error) {
	const prefix string = "Failed to find the latest:"
	if l.onProgress != nil {
		l.onProgress(fmt.Sprintln(prefix, err))
	} else {
		log.Println(prefix, err)
	}
}

func (l paperLoader) LoadLatest(f ProjectsCallback) {
	c := l.collector.Clone()
	c.OnError(l.logError)
	c.OnResponse(func(r *colly.Response) {
		var projects Projects
		if err := json.Unmarshal(r.Body, &projects); err != nil {
			log.Println("Failed to unmarshal projects:", err)
		} else {
			f(&projects.Projects)
		}
	})
	c.Visit(l.url.String())
}

func NewPaperLoader() PaperLoader {
	loader := &paperLoader{
		collector: colly.NewCollector(),
		url:       &url.URL{},
	}
	loader.collector.MaxBodySize = 0
	if parsedUrl, err := url.Parse("https://papermc.io/api/v2/projects"); err != nil {
		log.Println("Failed to parse URL", err)
	} else {
		loader.url = parsedUrl
	}
	return loader
}
