:::mermaid
flowchart LR
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

Jobs <--HTTP--> API;
API <--EF--> DB;

:::
