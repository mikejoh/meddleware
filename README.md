# mm

<p align="center">
<img src="https://github.com/mikejoh/mm/assets/899665/0158ee4b-e5b1-4f70-ae04-82a01217de34" alt="mm" />
</p>


`mm` - Metrics middleware for your Go HTTP Clients!

## Install

`go get github.com/mikejoh/mm`

## Usage

This example will output the client metrics at `http://localhost:8080` we get from the chained metric middlewares:
```
func main() {
    pr := prometheus.NewRegistry()

    mm := mm.New(pr, "namespace", "subsystem")

    c := http.DefaultClient
    c.Transport = mm.DefaultMiddlewares(http.DefaultTransport)
    c.Get("http://www.google.com")

    http.Handle("/metrics", promhttp.HandlerFor(mm.Registry, promhttp.HandlerOpts{}))
    http.ListenAndServe(":8080", nil)
}
```
