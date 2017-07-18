package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/go-hep/hplot"
)

func main() {
	fmt.Println("Rubik's Cube Solve Analyzer!")
	fmt.Println("Solve file should be formated with one solve per line.")
	fmt.Println("Substeps of solve can also be separated by commas.")
	fmt.Println("The default options for the following steps are show in []")
	fmt.Println("After inputin the filename, you can probably just hit enter.")
	fmt.Print("\nPlease input filename[]: ")
	var cmd string
	fmt.Scanln(&cmd)
	filename := cmd
	fmt.Println("Filename: ", filename)
	fmt.Print("Does each line have a trailing comma(yes for HARCS)([Y]/N): ")
	cmd = ""
	fmt.Scanln(&cmd)
	trailingComma := true
	if cmd != "" && strings.ToLower(string(cmd[0])) == "n" {
		trailingComma = false
	}
	if trailingComma {
		fmt.Println("Trailing comma: Yes")
	} else {
		fmt.Println("Trailing comma: No")
	}
	method := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	fmt.Print("What is the method name?[" + method + "]: ")
	cmd = ""
	fmt.Scanln(&cmd)
	if cmd != "" {
		method = cmd
	}
	fmt.Println("Method name: ", method)
	fmt.Println("The following formats are supported: eps, jpg, jpeg, pdf, png, svg, tif and tiff.")
	fmt.Print("Save Extension[png]: ")
	saveas := "png"
	cmd = ""
	fmt.Scanln(&cmd)
	if cmd != "" {
		saveas = cmd
	}
	fmt.Println("Save Extension: ", saveas)

	fmt.Println("Generating plots")
	processFile(filename, trailingComma, method, vg.Points(1000), saveas)
	fmt.Println("Plots generated and saved")
	fmt.Println("Press enter to close...")
	fmt.Scanln()
}

func processFile(filename string, trailingComma bool, method string, size vg.Length, saveas string) {
	f, err := os.Open(filename)
	defer f.Close()
	check(err)
	numSolves, err := lineCounter(f)
	check(err)
	_, err = f.Seek(0, 0)
	check(err)
	reader := bufio.NewReader(f)
	firstline, _, err := reader.ReadLine()
	check(err)
	numSteps := len(strings.Split(string(firstline), ","))
	if trailingComma {
		numSteps--
	}
	_, err = f.Seek(0, 0)
	check(err)
	reader = bufio.NewReader(f)

	var moves = make([]map[string]int, numSteps+1)
	for i := 0; i < numSteps+1; i++ {
		moves[i] = make(map[string]int)
	}

	movecounts := make([][]int, numSteps+1)
	for i := range movecounts {
		movecounts[i] = make([]int, numSolves)
	}
	re := regexp.MustCompile("[^a-zA-Z]+")

	for i := 0; i < numSolves; i++ {
		line, _, err := reader.ReadLine()
		check(err)
		substeps := strings.Split(string(line), ",")[0:numSteps]
		for j := 0; j < numSteps; j++ {
			step := strings.TrimSpace(substeps[j])
			if step == "" {
				movecounts[j][i] = 0
			} else {
				movecounts[j][i] = len(strings.Split(step, " "))
				movecounts[numSteps][i] += movecounts[j][i]
			}
			letters := re.ReplaceAllString(step, "")
			for _, char := range letters {
				moves[j][string(char)]++
				moves[numSteps][string(char)]++
			}
		}
	}

	tp, err := hplot.NewTiledPlot(draw.Tiles{Cols: numSteps+1, Rows: 1})
	if err != nil {
		check(err)
	}

	for num, step := range movecounts {
		m := mean(step)
		p := tp.Plot(0, num)
		if num == numSteps {
			p.Title.Text = "Overall"
		} else {
			p.Title.Text = "Step #" + strconv.Itoa(num+1)
		}
		if num == 0{
			p.Y.Label.Text = "Percentage of Solves"
		}
		// Create a histogram of our values drawn
		// from the standard normal.
		v := make(plotter.Values, len(step))
		for i, x := range step {
			v[i] = float64(x)
		}
		n := 0
		if max(step)-min(step) != 0 {
			n = max(step) - min(step) + 2
		}
		h, err := plotter.NewHist(v, n)
		if err != nil {
			check(err)
		}
		// Normalize the area under the histogram to
		// sum to one.
		h.Normalize(100)
		h.Color = color.RGBA{R: 31, G: 119, B: 180, A: 255}
		h.FillColor = color.RGBA{R: 31, G: 119, B: 180, A: 255}
		p.Add(h)

		// The normal distribution function
		avg := make(plotter.XYs, 2)
		avg[0].X = m
		avg[0].Y = 0
		avg[1].X = m
		avg[1].Y = 100
		l, err := plotter.NewLine(avg)
		if err != nil {
			check(err)
		}
		l.Color = color.RGBA{R: 255, A: 255}
		l.LineStyle.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		p.Add(l)
		p.Add(plotter.NewGrid())
		p.Legend.Add("Mean", l)
		p.Legend.Top = true
		p.Legend.Left = false
	}

	// Save the plot to a PNG file.
	if err := tp.Save(size, -1, strings.Replace(method + "_Move_Count_Distribution."+saveas, " ", "_", -1)); err != nil {
		check(err)
	}

	tp2, err := hplot.NewTiledPlot(draw.Tiles{Cols: numSteps+1, Rows: 1})
	if err != nil {
		check(err)
	}

	for num, _ := range movecounts{

		var sum float64
		reversemap := make(map[int]string, len(moves[num]))
		for k, v := range moves[num] {
			reversemap[v] = k
			sum += float64(v)
		}

		keys := make([]int, len(reversemap))
		var i int
		for k := range reversemap {
			keys[i] = k
			i++
		}

		p := tp2.Plot(0, num)
		if num == numSteps {
			p.Title.Text = "Overall"
		} else {
			p.Title.Text = "Step #" + strconv.Itoa(num+1)
		}
		if num == 0{
			p.Y.Label.Text = "Percentage of Moves"
		}
		p.Y.Max = 100
		p.Y.Min = 0
		p.Legend.Top = true

		sortedKeys := bubbleSort(keys)
		names := make([]string, len(reversemap))
		group := make(plotter.Values, len(reversemap))
		for i, k := range sortedKeys {
			group[i] = float64(k) * 100 / sum
			names[i] = reversemap[k]
		}
		bars, err := plotter.NewBarChart(group, size/vg.Points(float64(len(reversemap)*(numSteps+1))))
		if err != nil {
			check(err)
		}
		bars.LineStyle.Width = vg.Length(0)
		bars.Color = color.RGBA{R: 31, G: 119, B: 180, A: 255}
		p.Add(bars)
		p.Add(plotter.NewGrid())
		p.NominalX(names...)
	}

	// Save the plot to a PNG file.
	if err := tp2.Save(size, -1, strings.Replace(method + "_Move_Distribution."+saveas, " ", "_", -1)); err != nil {
		check(err)
	}
}

func bubbleSort(arr []int) []int {
	for i := 1; i < len(arr); i++ {
		for j := 0; j < len(arr)-i; j++ {
			if arr[j] < arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
	return arr
}

func mean(x []int) float64 {
	var sum float64
	for _, i := range x {
		sum += float64(i)
	}
	return sum / float64(len(x))
}

func min(x []int) int {
	m := x[0]
	for _, i := range x {
		if i < m {
			m = i
		}
	}
	return m
}

func max(x []int) int {
	m := x[0]
	for _, i := range x {
		if i > m {
			m = i
		}
	}
	return m
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

func check(e error) {
	if e != nil {
		fmt.Println(e)
		fmt.Println("Press enter to close...")
		fmt.Scanln()
		os.Exit(2)
	}
}
