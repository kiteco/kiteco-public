package rundb

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	initString = `<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
<script type="text/javascript">
// Load google charts
google.charts.load('current', {'packages':['corechart']});
// Draw the chart and set the chart values
function drawPieChart(data, options, divId) {
  return function() {
	  var chart = new google.visualization.PieChart(document.getElementById(divId));
	  chart.draw(google.visualization.arrayToDataTable(data), options);
	}
}
</script>`

	pieChartTemplate = `
<div id="{{.DivID}}" style="display: inline-block"></div>
<script type="text/javascript">

	var data = [['Label', 'Count'],{{range $index, $v := .Values}}{{if $index}},{{end}}['{{$v.Label}}',{{$v.Value}}]{{end}}];

	// Optional; add a title and set the width and height of the chart
	var options = {'title':"{{.Title}}", 'width':{{.Width}}, 'height':{{.Height}}};
	google.charts.setOnLoadCallback(drawPieChart(data, options, '{{.DivID}}'));

</script>`
)

// PieChartDrawer generates string to use in RunDB result to represent PieChart
type PieChartDrawer struct {
	nextID       int
	template     *template.Template
	ChartPerLine int
}

// NewPieChartDrawer create a pie chart drawer it can't be shared between multiple RunDB result (the initialization
// string must be added exactly once per web page) and only one should be used for one RunDB result
func NewPieChartDrawer(chartPerLine int) (*PieChartDrawer, error) {
	theTemplate, err := template.New("pieChartTemplate").Parse(pieChartTemplate)
	if err != nil {
		return nil, err
	}
	return &PieChartDrawer{template: theTemplate,
		ChartPerLine: chartPerLine}, nil
}

// PieChartData is the struct used to represent the data for drawing a pie chart
type PieChartData struct {
	Label string
	Value float64
}

// GetPieChartString returns the string to use as the value for a Result in a RunDB result
// The PieChartData will be automatically sorted to enforce that 2 pie charts with the same labels will have the
// same color set.
func (pcd *PieChartDrawer) GetPieChartString(data []PieChartData, width, height int, title string) (string, error) {
	var result string
	if pcd.nextID == 0 {
		result += initString
	}

	var buffer bytes.Buffer
	templateData := struct {
		Title  string
		Height int
		Width  int
		Values []PieChartData
		DivID  string
	}{
		Title:  title,
		Height: height,
		Width:  width,
		DivID:  fmt.Sprintf("pieChart%d", pcd.nextID),
		Values: data,
	}
	pcd.nextID++
	if err := pcd.template.Execute(&buffer, templateData); err != nil {
		return "", err
	}

	result += buffer.String()
	if pcd.nextID%pcd.ChartPerLine == 0 {
		result += "<br/>"
	}
	return result, nil
}
