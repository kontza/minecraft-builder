package paper_loader

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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
	LoadProject(project string)
}

type paperLoader struct {
	url       *url.URL
	collector *colly.Collector
}

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	log.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	log.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func DownloadFile(filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	log.Println()

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}

func (l paperLoader) download(project string, version string, build int, artifact string) {
	log.Println("Artifact to load", artifact)
	// https://papermc.io/api/v2/projects/paper/versions/1.18.2/builds/277/downloads/paper-1.18.2-277.jar
	sourceUrl := fmt.Sprintf("%s/%s/versions/%s/builds/%d/downloads/%s", l.url.String(), project, version, build, artifact)
	DownloadFile(artifact, sourceUrl)
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
	c.Visit(fmt.Sprintf("%s/%s/versions/%s/builds/%d", l.url.String(), project, version, buildNumber))
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
	c.Visit(fmt.Sprintf("%s/%s/versions/%s", l.url.String(), project, version))
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
	c.Visit(fmt.Sprintf("%s/%s", l.url.String(), project))
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
	c.Visit(l.url.String())
}

func NewPaperLoader() PaperLoader {
	loader := &paperLoader{
		collector: colly.NewCollector(),
		url:       &url.URL{},
	}
	if parsedUrl, err := url.Parse("https://papermc.io/api/v2/projects"); err != nil {
		log.Println("Failed to parse URL", err)
	} else {
		loader.url = parsedUrl
	}
	loader.collector.OnError(func(r *colly.Response, err error) {
		log.Println("Failed to find the latest:", err)
	})
	loader.collector.OnScraped(func(r *colly.Response) {
		log.Println("Scrape done.", r.Request.URL)
	})
	return loader
}
