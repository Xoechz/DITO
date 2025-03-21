:::mermaid
flowchart TB
subgraph Internal
API;
Jobs;
DB[(Data)];
end
subgraph External
eAPI[API];
eDB[(Data)];
end

eAPI --HTTP--> API;
eDB --EF--> Jobs;

API --HTTP--> Jobs;
API <--EF--> DB;

Jobs --CSV--> Result[Result File];

subgraph OTEL Collector
Process[Processing<br>----<br>Transform traces into data point specific traces and metrics]
Sampling
Exporter
end

Jobs -. Job Traces .- Process;
API -. Job Traces .- Process;

Process -. Data Traces .- Sampling;
Process -. Metrics .- Exporter;
Sampling -..- Exporter;

:::
