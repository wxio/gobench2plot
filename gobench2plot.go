package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var gauge *bool = flag.Bool("gauge", false, "Provided two benchmark files <old> <new> output the ratio difference")

type benchdata struct {
	nsMap           map[string]string
	allocedBytesMap map[string]string
	allocedMap      map[string]string
	mbMap           map[string]string
}

// Examples:
// BenchmarkParsePage        100      17788153 ns/op
// BenchmarkWrite_4KB_WithIndex       50000         61010 ns/op      67.14 MB/s         598 B/op         16 allocs/op
//
// using b.Run
// BenchmarkQuery/q1.1-6         	   10000	    272723 ns/op
var (
	nsExp           *regexp.Regexp = regexp.MustCompile("^(Benchmark.+?)\\s+.*?(\\S+) ns/op.*")
	allocedBytesExp *regexp.Regexp = regexp.MustCompile("^(Benchmark.+?)\\s+.*?(\\S+) B/op.*")
	allocedExp      *regexp.Regexp = regexp.MustCompile("^(Benchmark.+?)\\s+.*?(\\S+) allocs/op.*")
	mbExp           *regexp.Regexp = regexp.MustCompile("^(Benchmark.+?)\\s+.*?(\\S+) MB/s.*")
)

func main() {
	flag.Parse()

	if *gauge {
		if len(flag.Args()) != 2 {
			fmt.Fprintf(os.Stderr, "-gauge requires two args <old file> <new file>. %v\n", os.Args)
			os.Exit(1)
		}
		oldf, err1 := os.Open(flag.Args()[0])
		newf, err2 := os.Open(flag.Args()[1])
		if err1 != nil {
			fmt.Fprintf(os.Stderr, "file error %v\n", err1)
			os.Exit(1)
		}
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "file error %v\n", err2)
			os.Exit(1)
		}
		bdold := newBenchdata(oldf)
		bdnew := newBenchdata(newf)
		diff := &benchdata{}
		diff.nsMap = diffMap(bdold.nsMap, bdnew.nsMap)
		diff.allocedBytesMap = diffMap(bdold.allocedBytesMap, bdnew.allocedBytesMap)
		diff.allocedMap = diffMap(bdold.allocedMap, bdnew.allocedMap)
		diff.mbMap = diffMap(bdold.mbMap, bdnew.mbMap)
		diff.writeDiffBenchmark()
	} else {
		if len(flag.Args()) != 0 {
			fmt.Fprintf(os.Stderr, "command line arguments provided, but only accepts stdin\n")
			os.Exit(1)
		}
		bd := newBenchdata(os.Stdin)
		bd.writeSingleBenchmark()
	}
}

func diffMap(oldmap, newmap map[string]string) map[string]string {
	diff := make(map[string]string)
	for key, value := range newmap {
		oldv, exist := oldmap[key]
		if exist {
			nv, err := strconv.Atoi(value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "NaN <new> key:%s error:%v\n", key, err)
				continue
			}
			ov, err := strconv.Atoi(oldv)
			if err != nil {
				fmt.Fprintf(os.Stderr, "NaN <old> key:%s error:%v\n", key, err)
				continue
			}
			var ratio float32 = float32(ov) / float32(nv)
			diff[key] = fmt.Sprintf("%7.2f", ratio)
		}
	}
	return diff
}

func (bd *benchdata) writeDiffBenchmark() {
	fmt.Printf("<Benchmarks>\n")
	fmt.Printf(" <TimeRation>\n")
	for key := range bd.nsMap {
		fmt.Printf("  <%[1]v_time>%[2]v</%[1]v_time>\n", key, bd.nsMap[key])
	}
	fmt.Printf(" </TimeRation>\n")

	fmt.Printf(" <AllocsBytesRatio>\n")
	for key := range bd.allocedBytesMap {
		fmt.Printf("  <%[1]v_bytes>%[2]v</%[1]v_bytes>\n", key, bd.allocedBytesMap[key])
	}
	fmt.Printf(" </AllocsBytesRatio>\n")

	fmt.Printf(" <AllocsRatio>\n")
	for key := range bd.allocedMap {
		fmt.Printf("  <%[1]v_allocs>%[2]v</%[1]v_allocs>\n", key, bd.allocedMap[key])
	}
	fmt.Printf(" </AllocsRatio>\n")

	fmt.Printf(" <mbRatio>\n")
	for key := range bd.mbMap {
		fmt.Printf("  <%[1]v_mbpers>%[2]v</%[1]v_mbpers>\n", key, bd.mbMap[key])
	}
	fmt.Printf(" </mbRatio>\n")
	fmt.Printf("</Benchmarks>\n")
}

func (bd *benchdata) writeSingleBenchmark() {
	fmt.Printf("<Benchmarks>\n")
	fmt.Printf(" <NsPerOp>\n")
	for key := range bd.nsMap {
		fmt.Printf("  <%[1]v_time>%[2]v</%[1]v_time>\n", key, bd.nsMap[key])
	}
	fmt.Printf(" </NsPerOp>\n")

	fmt.Printf(" <AllocsBytesPerOp>\n")
	for key := range bd.allocedBytesMap {
		fmt.Printf("  <%[1]v_bytes>%[2]v</%[1]v_bytes>\n", key, bd.allocedBytesMap[key])
	}
	fmt.Printf(" </AllocsBytesPerOp>\n")

	fmt.Printf(" <AllocsPerOp>\n")
	for key := range bd.allocedMap {
		fmt.Printf("  <%[1]v_allocs>%[2]v</%[1]v_allocs>\n", key, bd.allocedMap[key])
	}
	fmt.Printf(" </AllocsPerOp>\n")

	fmt.Printf(" <mbPerSec>\n")
	for key := range bd.mbMap {
		fmt.Printf("  <%[1]v_mbpers>%[2]v</%[1]v_mbpers>\n", key, bd.mbMap[key])
	}
	fmt.Printf(" </mbPerSec>\n")
	fmt.Printf("</Benchmarks>\n")
}

func newBenchdata(in io.Reader) *benchdata {
	bd := &benchdata{
		nsMap:           make(map[string]string),
		allocedBytesMap: make(map[string]string),
		allocedMap:      make(map[string]string),
		mbMap:           make(map[string]string),
	}
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if tokens := nsExp.FindStringSubmatch(line); tokens != nil {
			key := strings.Replace(tokens[1], "/", "_", -1)
			bd.nsMap[key] = tokens[2]
		}
		if tokens := allocedBytesExp.FindStringSubmatch(line); tokens != nil {
			key := strings.Replace(tokens[1], "/", "_", -1)
			bd.allocedBytesMap[key] = tokens[2]
		}
		if tokens := allocedExp.FindStringSubmatch(line); tokens != nil {
			key := strings.Replace(tokens[1], "/", "_", -1)
			bd.allocedMap[key] = tokens[2]
		}
		if tokens := mbExp.FindStringSubmatch(line); tokens != nil {
			key := strings.Replace(tokens[1], "/", "_", -1)
			bd.mbMap[key] = tokens[2]
		}
	}
	return bd
}
