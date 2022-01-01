package rundb

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"text/template"
)

const (
	histInitString = `<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
<script type="text/javascript">
// Load google charts
google.charts.load('current', {'packages':['corechart']});
// Draw the chart and set the chart values
function drawChart(data, options, divId) {
  return function() {
	  var chart = new google.visualization.Histogram(document.getElementById(divId));
	  chart.draw(google.visualization.arrayToDataTable(data), options);
	}
}
</script>`

	histogramTemplate = `
<div id="{{.DivID}}" style="display: inline-block"></div>
<script type="text/javascript">

	var data = [['Name', 'Number']{{range $index, $v := .Values}},['delta',{{$v}}]{{end}}];

	// Optional; add a title and set the width and height of the chart
	var options = {'title':"{{.Title}}", 'width':{{.Width}}, 'height':{{.Height}}, 
					histogram: {
						minValue: {{.Min}},
						maxValue: {{.Max}}
					}};
	google.charts.setOnLoadCallback(drawChart(data, options, '{{.DivID}}'));

</script>`

	multiSeriesHistTemplate = `
<div id="{{.DivID}}" style="display: inline-block"></div>
<script type="text/javascript">

	var data = [[{{range $index, $v := .Labels}}{{if gt $index 0}},{{end}}"{{$v}}"{{end}}]{{range $index, $v := .Values}},[{{$v}}]{{end}}];

	// Optional; add a title and set the width and height of the chart
	var options = {'title':"{{.Title}}", 'width':{{.Width}}, 'height':{{.Height}}, 
					histogram: {
						minValue: {{.Min}},
						maxValue: {{.Max}}
					},
					interpolateNulls: false
				};
	google.charts.setOnLoadCallback(drawChart(data, options, '{{.DivID}}'));

</script>`
)

// HistogramDrawer generates string to use in RunDB result to represent PieChart
type HistogramDrawer struct {
	nextID              int
	template            *template.Template
	multiSeriesTemplate *template.Template
}

// NewHistogramDrawer create a pie chart drawer it can't be shared between multiple RunDB result (the initialization
// string must be added exactly once per web page) and only one should be used for one RunDB result
func NewHistogramDrawer() (*HistogramDrawer, error) {
	theTemplate, err := template.New("histogramTemplate").Parse(histogramTemplate)
	if err != nil {
		return nil, err
	}
	theMultiSeriesTemplate, err := template.New("multiSeriesHistTemplate").Parse(multiSeriesHistTemplate)
	if err != nil {
		return nil, err
	}
	return &HistogramDrawer{template: theTemplate, multiSeriesTemplate: theMultiSeriesTemplate}, nil
}

// HistogramData is the struct used to represent the data for drawing a pie chart
type HistogramData float64

// MultiSeriesHistogramData is a string reprensenting a set of value (one for each serie). Don't use this struct
// It is only exported for the needs of the template. Call GetMultiSeriesHistogramString, it will take care of
// generating the required data for the template
type MultiSeriesHistogramData string

// GetHistogramDataFromFrequencyMap ...
func GetHistogramDataFromFrequencyMap(frequencies map[int]int) ([]HistogramData, int, int) {
	var result []HistogramData
	var minSet bool
	var min, max int
	for v, count := range frequencies {
		if !minSet {
			min = v
			max = v
			minSet = true
		}
		min = int(math.Min(float64(min), float64(v)) - 1)
		max = int(math.Max(float64(max), float64(v)) + 1)

		for i := 0; i < count; i++ {
			result = append(result, HistogramData(v))
		}
	}
	return result, min, max
}

// GetHistogramString returns the string to use as the value for a Result in a RunDB result
func (pcd *HistogramDrawer) GetHistogramString(data []HistogramData, width, height int, title string, min, max float32) (string, error) {
	var result string
	if pcd.nextID == 0 {
		result += histInitString
	}

	var buffer bytes.Buffer
	templateData := struct {
		Title  string
		Height int
		Width  int
		Min    int
		Max    int
		Values []HistogramData
		DivID  string
	}{
		Title:  title,
		Height: height,
		Width:  width,
		DivID:  fmt.Sprintf("histogram_%d", pcd.nextID),
		Values: data,
		Min:    int(math.Ceil(float64(min - 1))),
		Max:    int(math.Ceil(float64(max + 1))),
	}
	pcd.nextID++
	if err := pcd.template.Execute(&buffer, templateData); err != nil {
		return "", err
	}

	result += buffer.String()
	result += "<br/>"

	return result, nil
}

// GetMultiSeriesHistogramString build an histogram for multiple series
// The first dimension of data should be the number of series, second dimension containing the datapoint for each series
// labels is for the names of each series (so len(data) == len(labels)
func (pcd *HistogramDrawer) GetMultiSeriesHistogramString(data [][]HistogramData, labels []string, width, height int, title string, min, max float32) (string, error) {
	var result string
	if pcd.nextID == 0 {
		result += histInitString
	}

	multiSerieData := makeMultiSeriesData(data)
	var buffer bytes.Buffer
	templateData := struct {
		Title  string
		Height int
		Width  int
		Min    int
		Max    int
		Values []MultiSeriesHistogramData
		Labels []string
		DivID  string
	}{
		Title:  title,
		Height: height,
		Width:  width,
		DivID:  fmt.Sprintf("histogram_%d", pcd.nextID),
		Values: multiSerieData,
		Labels: labels,
		Min:    int(math.Ceil(float64(min - 1))),
		Max:    int(math.Ceil(float64(max + 1))),
	}
	pcd.nextID++
	if err := pcd.multiSeriesTemplate.Execute(&buffer, templateData); err != nil {
		return "", err
	}
	result += buffer.String()
	result += "<br/>"

	return result, nil
}

// GetTwoSeriesHistogramString returns the string to use as the value for a Result in a RunDB result
// labels should be an array of 0 (negative sample) and 1 (positive samples) of the same lengths than data
func (pcd *HistogramDrawer) GetTwoSeriesHistogramString(data []HistogramData, labels []float32, width, height int, title string, min, max float32) (string, error) {
	twoSerieData := makeTwoSeriesData(data, labels)
	return pcd.GetMultiSeriesHistogramString(twoSerieData, []string{"Negative", "Positive"}, width, height, title, min, max)
}

func makeTwoSeriesData(data []HistogramData, labels []float32) [][]HistogramData {
	result := make([][]HistogramData, 2)

	for i, d := range data {

		if labels[i] == 0 {
			result[0] = append(result[0], d)
		} else {
			result[1] = append(result[1], d)
		}
	}
	return result
}

func makeMultiSeriesData(data [][]HistogramData) []MultiSeriesHistogramData {
	var result []MultiSeriesHistogramData
	maxLength := len(data[0])
	for _, sl := range data {
		if len(sl) > maxLength {
			maxLength = len(sl)
		}
	}

	for i := 0; i < maxLength; i++ {
		var line []string
		for _, d := range data {
			if len(d) > i {
				line = append(line, fmt.Sprint(d[i]))
			} else {
				line = append(line, "null")
			}
		}
		result = append(result, MultiSeriesHistogramData(strings.Join(line, ",")))
	}
	return result
}
