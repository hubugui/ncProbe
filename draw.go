package main

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

var _g_taskEventMap = make(map[string][]TaskEvent)

// generate random data for line chart
func generateLineItems() []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < 7; i++ {
		items = append(items, opts.LineData{Value: rand.Intn(300)})
	}
	return items
}

func klineDataZoomBoth() *charts.Kline {
	kline := charts.NewKLine()

	x := make([]string, 0)
	y := make([]opts.KlineData, 0)
	// for i := 0; i < len(kd); i++ {
	// 	x = append(x, kd[i].date)
	// 	y = append(y, opts.KlineData{Value: kd[i].data})
	// }

	for taskName, taskEventSlice := range _g_taskEventMap {
		fmt.Printf("taskName=%s\n", taskName)
	    for index, taskEvt := range taskEventSlice { 
	        fmt.Sprintf("%s,%d,%s,%d\n", taskEvt.timestamp, taskEvt.eventType, taskEvt.event, index + 1)

	        data := [4]int{1, 
	        				2, 
	        				3, 
	        				4}

			x = append(x, taskEvt.timestamp)
			y = append(y, opts.KlineData{Value: data})
	    }
	}

	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "basic line example", 
		}),
		//charts.WithTitleOpts(opts.Title{Title: "DataZoom(inside&slider)", Subtitle: "1: unhealth, 2: health, 3: restart"}),
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 20,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Scale: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "slider",
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
	)

	kline.SetXAxis(x).AddSeries("kline", y)
	return kline
}

func httpserverKline(w http.ResponseWriter, _ *http.Request) {
	// create a new line instance
	line := klineDataZoomBoth()
	line.Render(w)
}

func httpserver(w http.ResponseWriter, _ *http.Request) {
	// create a new line instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title: "Nomad Task Health Event(inside&slider)",
			Subtitle: "1: unhealth, 2: health, 3: restart",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 100,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Scale: true,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "slider",
			Start:      50,
			End:        100,
			XAxisIndex: []int{0},
		}),
	)

	x := make([]string, 0)
	y := make([]opts.LineData, 0)
	for taskName, taskEventSlice := range _g_taskEventMap {
		fmt.Printf("taskName=%s\n", taskName)
	    for index, taskEvt := range taskEventSlice { 
	        fmt.Sprintf("%s,%d,%s,%d\n", taskEvt.timestamp, 
	        							taskEvt.eventType, 
	        							taskEvt.event, 
	        							index + 1)

			x = append(x, taskEvt.timestamp)
			y = append(y, opts.LineData{Value: taskEvt.eventType})
	    }
	}

	// Put data into instance
	line.SetXAxis(x).
		AddSeries("Category A", y).
		SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	line.Render(w)
}

func draw(taskEventMap map[string][]TaskEvent, port int) {
	_g_taskEventMap = taskEventMap

	fmt.Printf("please open \"http://localhost:%d\" to view results\n", port)

	http.HandleFunc("/", httpserver)
	http.HandleFunc("/kline", httpserverKline)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}