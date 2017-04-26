# Graphite-render-exporter
Export performance metrics from Graphite as Prometheus metrics

#### Why?
Sometimes I have to run two monitoring solutions in parallel. Graphite infrastricture provides some performance metics and I want them to be stored in Prometheus.

#### Usage
Quick run:
        go run graphite-render-exporter.go

Recommended usage is docker container:
        docker run mbarzilovich/graphite-render-exporter

#### Configuration
Application is configured with environment variables
+ GRAPHITE_URL = "http://localhost:8080/render"         URL or Graphite render API
+ TARGETS = "\*.\*"                                     Globe to select metrics in Gripthite render API
+ POLL_DEPTH = "50s"                                    Time back from now in Graphite render API request ("from=-50s" parameter)
+ HTTP_USER = ""                                        Basic authentication user. No basic auth if empty
+ HTTP_PASSWORD = ""                                    Basic authentication password. No basic auth if empty
        
        docker run -d -name graphite-render-exporter -e GRAPHITE_URL="http://example.com/metrics" -e TARGETS="metrics.cpu.*" -e POLL_DEPTH=4min -e HTTP_USER=myuser -e HTTP_PASSWORD=mypassword -p 8081:8081 mbarzilovich/graphite-render-exporter
        
        
        
        
        