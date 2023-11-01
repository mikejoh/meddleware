# mm

`mm` - Metrics middleware for your Go HTTP Clients!

## Install

`go get github.com/mikejoh/mm`

## Usage

This example will output the client metrics at `http://Localhost:8080` we get from the chained metric middlewares:
```
func main() {
    pr := prometheus.NewRegistry()
	mm := New(pr, "namespace", "subsystem")
	c := http.DefaultClient
	c.Transport = mm.DefaultMiddlewares(http.DefaultTransport)

	c.Get("http://www.google.com")
	http.Handle("/metrics", promhttp.HandlerFor(mm.Registry, promhttp.HandlerOpts{}))
	http.ListenAndServe(":8080", nil)
}
```
