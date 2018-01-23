package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Repo struct {
	Name        string
	URL         string
	Description string
	Complexity  int
}

var Repos = []Repo{
	{
		Name:        "cangallo",
		URL:         "git://github.com/jfontan/cangallo.git",
		Description: "Small repository that should be fast to clone",
		Complexity:  0,
	},
	{
		Name:        "octoprint-tft",
		URL:         "git://github.com/mcuadros/OctoPrint-TFT.git",
		Description: "Small repository that should be fast to clone",
		Complexity:  0,
	},
	{
		Name:        "numpy",
		URL:         "git://github.com/numpy/numpy.git",
		Description: "Average repository",
		Complexity:  1,
	},
	{
		Name:        "tensorflow",
		URL:         "git://github.com/tensorflow/tensorflow.git",
		Description: "Average repository",
		Complexity:  1,
	},
	// {
	// 	Name:        "pandas",
	// 	URL:         "git://github.com/pandas-dev/pandas.git",
	// 	Description: "Average repository",
	// 	Complexity:  1,
	// },
	{
		Name:        "upsilon",
		URL:         "git://github.com/upsilonproject/upsilon-common.git",
		Description: "Average repository",
		Complexity:  1,
	},
	{
		Name:        "bismuth",
		URL:         "git://github.com/hclivess/Bismuth.git",
		Description: "Big files repo (100Mb)",
		Complexity:  2,
	},
}

type BenchData struct {
	MemStart runtime.MemStats
	MemMax   runtime.MemStats
	Duration time.Duration
}

func getMaxMem(tmp, max *runtime.MemStats) {
	runtime.ReadMemStats(tmp)
	if tmp.HeapAlloc > max.HeapAlloc {
		// println("max", max.HeapAlloc, tmp.HeapAlloc)
		*tmp, *max = *max, *tmp
	}
}

func bench(f func() error) (BenchData, error) {
	ticker := time.NewTicker(time.Millisecond * 100).C
	done := make(chan bool)
	tStart := time.Now()

	mStart := &runtime.MemStats{}
	runtime.ReadMemStats(mStart)

	mMax := &runtime.MemStats{}
	mTmp := &runtime.MemStats{}

	go func() {
		for {
			select {
			case <-done:
				// collect mem data one last time
				getMaxMem(mTmp, mMax)
				done <- true
				return
			case <-ticker:
				getMaxMem(mTmp, mMax)
			}
		}
	}()

	err := f()

	// send done signal to memory collector for it to finish
	done <- true
	<-done

	return BenchData{
		Duration: time.Since(tStart),
		MemStart: *mStart,
		MemMax:   *mMax}, err
}

func CloneRepo(url, dir string) (BenchData, *git.Repository, error) {
	var r *git.Repository

	os.RemoveAll(dir)

	b, err := bench(func() error {
		var err error
		r, err = git.PlainClone(dir, false, &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})

		return err
	})

	return b, r, err
}

func PushLocal(from, to string) (BenchData, *git.Repository, error) {
	var r *git.Repository

	os.RemoveAll(to)

	b, err := bench(func() error {
		fromRepo, err := git.PlainOpen(from)
		if err != nil {
			return err
		}

		r = fromRepo

		_, err = git.PlainInit(to, false)
		if err != nil {
			return err
		}

		_, err = fromRepo.CreateRemote(&config.RemoteConfig{
			Name: "benchPush",
			URLs: []string{to},
		})

		rspec := make([]config.RefSpec, 0)

		refIter, err := fromRepo.References()
		refIter.ForEach(func(r *plumbing.Reference) error {
			rs := fmt.Sprintf("+%s:%s/%s", r.Name(), r.Name(), "bench")
			rspec = append(rspec, config.RefSpec(rs))

			return nil
		})

		err = fromRepo.Push(&git.PushOptions{
			RemoteName: "benchPush",
			RefSpecs:   rspec,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return b, r, err
}

func PrintResult(name string, repeat int, step string, text string) {
	fmt.Printf("%v(%v), %v, %v\n", name, repeat, step, text)
}

func PrintBench(name string, repeat int, step string, b BenchData) {
	text := fmt.Sprintf("Time: %v, HeapAlloc: %v, HeapAllocTotal: %v",
		b.Duration,
		b.MemMax.HeapAlloc-b.MemStart.HeapAlloc,
		b.MemMax.HeapAlloc,
	)

	PrintResult(name, repeat, step, text)
}

func PrintError(name string, repeat int, step string, e error) {
	text := fmt.Sprintf("Error: %v", e)

	PrintResult(name, repeat, step, text)
}

func RunStep(name string, repeat int, step string, f func() (BenchData, error)) {
	for i := 0; i < repeat; i++ {
		runtime.GC()
		bench, err := f()

		if err != nil {
			PrintError(name, i, step, err)
		} else {
			PrintBench(name, i, step, bench)
		}
	}
}

func RunBenchmark(repos []Repo, complexity int, repeat int) error {
	for _, r := range repos {
		if r.Complexity > complexity {
			continue
		}

		var repo *git.Repository
		// var bench BenchData
		// var err error

		step := "clone remote"

		RunStep(r.Name, repeat, step, func() (BenchData, error) {
			bench, tmpRepo, err := CloneRepo(r.URL, "./repo/")
			repo = tmpRepo

			return bench, err
		})

		step = "clone local"

		RunStep(r.Name, repeat, step, func() (BenchData, error) {
			bench, _, err := CloneRepo("./repo/", "./repo.clone/")

			return bench, err
		})

		step = "push local"

		RunStep(r.Name, repeat, step, func() (BenchData, error) {
			bench, _, err := PushLocal("./repo/", "./repo.push/")

			return bench, err
		})
	}

	return nil
}

// func

func main() {
	RunBenchmark(Repos, 3, 2)
	// RunBenchmark(Repos, 0, 2)
}
