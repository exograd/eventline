"use strict";

evOnPageLoaded("jobs", evSetupJobList);
evOnPageLoaded("jobs", evSetupAutoRefresh);
evOnPageLoaded("job_timeline", evSetupJobTimeline);
evOnPageLoaded("job_timeline", evSetupAutoRefresh);
evOnPageLoaded("job_metrics", evSetupJobMetrics);

function evSetupJobList() {
  const disableLinkSelector = "#ev-jobs a[data-action='disable']:not(.ev-disabled)";
  const disableLinks = document.querySelectorAll(disableLinkSelector);
  disableLinks.forEach(link => {
    link.onclick = evOnDisableJobClicked;
  });

  const enableLinkSelector = "#ev-jobs a[data-action='enable']:not(.ev-disabled)";
  const enableLinks = document.querySelectorAll(enableLinkSelector);
  enableLinks.forEach(link => {
    link.onclick = evOnEnableJobClicked;
  });

  const addFavouriteIconSelector = "#ev-jobs .icon.ev-is-favourite";
  const addFavouriteIcons = document.querySelectorAll(addFavouriteIconSelector);
  addFavouriteIcons.forEach(icon => {
    icon.onclick = evOnRemoveFavouriteJobClicked;
  });

  const removeFavouriteIconSelector = "#ev-jobs .icon:not(.ev-is-favourite)";
  const removeFavouriteIcons = document.querySelectorAll(removeFavouriteIconSelector);
  removeFavouriteIcons.forEach(icon => {
    icon.onclick = evOnAddFavouriteJobClicked;
  });
}

function evOnAddFavouriteJobClicked(event) {
  const icon = event.target.closest("span");
  const id = icon.dataset.id;

  const uri = `/jobs/id/${id}/add_favourite`
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
    .then(response => {
      location.reload();
    })
    .catch (e => {
      evShowError(`cannot add job to favourites: ${e.message}`);
    });
}

function evOnRemoveFavouriteJobClicked(event) {
  const icon = event.target.closest("span");
  const id = icon.dataset.id;

  const uri = `/jobs/id/${id}/remove_favourite`
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
    .then(response => {
      location.reload();
    })
    .catch (e => {
      evShowError(`cannot remove job from favourites: ${e.message}`);
    });
}

function evOnDisableJobClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;

  const uri = `/jobs/id/${id}/disable`
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
    .then(response => {
      location.reload();
    })
    .catch (e => {
      evShowError(`cannot disable job: ${e.message}`);
    });
}

function evOnEnableJobClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;

  const uri = `/jobs/id/${id}/enable`
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
    .then(response => {
      location.reload();
    })
    .catch (e => {
      evShowError(`cannot enable job: ${e.message}`);
    });
}

function evSetupJobTimeline() {
  const abortLinkSelector =
        "#ev-job-executions a[data-action='abort']:not(.ev-disabled)";
  const abortLinks = document.querySelectorAll(abortLinkSelector);
  abortLinks.forEach(link => {
    link.onclick = evOnAbortJobExecutionClicked;
  });

  const restartLinkSelector =
        "#ev-job-executions a[data-action='restart']:not(.ev-disabled)";
  const restartLinks = document.querySelectorAll(restartLinkSelector);
  restartLinks.forEach(link => {
    link.onclick = evOnRestartJobExecutionClicked;
  });
}

function evSetupJobMetrics() {
  const element = document.getElementById("job-metrics");
  const jobId = element.dataset.jobId;

  evSetupJobStatusCountMetrics(jobId);
  evSetupJobRunningTimeMetrics(jobId);
}

function evSetupJobStatusCountMetrics(jobId) {
  const element = document.getElementById("status-count-metrics");

  // Graph
  const graph = new EvTimeGraph(element, {
    yAxisTitle: "Job count",

    legend: [
      {class: "is-success", label: "successful"},
      {class: "is-warning", label: "aborted"},
      {class: "is-danger", label: "failed"},
    ],

    dataURI: function() {
      let granularity = "hour";
      if (this.timeRange == "30d") {
        granularity = "day";
      }

      const query = new URLSearchParams([
        ["start", evDateToUnix(this.startDate)],
        ["end", evDateToUnix(this.endDate)],
        ["granularity", granularity],
      ]);

      return `/jobs/id/${jobId}/metrics/status_counts?${query}`;
    },

    updateYScale: function (points) {
      if (!points) { points = []; }

      const counts = points.map(p => p[1]);
      const maxY = (counts.length>0) ? d3.max(counts) : 100;

      this.yTicks = d3.range(0, maxY, Math.ceil(maxY/6)).concat(maxY);
      this.yTickFormat = (d, i) => { return ""+d };
      this.yScale = d3.scaleLinear()
                      .domain([0, maxY])
                      .range([this.height, 0]);
    },

    renderData: function (points) {
      if (!points) { points = []; }

      points.forEach((p) => p[0] = new Date(p[0]*1000));
      points.sort((a, b) => a[0] - b[0]);

      // Tooltips
      onMouseover = function (event, p) {
        d3.select(this)
          .classed("is-active", true)
          .attr("r", 5);
      };

      onMouseout = function (event, p) {
        d3.select(this)
          .classed("is-active", false)
          .attr("r", 2.5);
      };

      // Bars
      let interval = this.xScale(this.xTicks[1]) - this.xScale(this.xTicks[0]);
      if (this.timeRange == "7d") {
        interval = interval / 24.0;
      }

      const barWidth = 0.8 * interval;
      const barPadding = 0.1 * interval;

      renderGroup = (name, className, index) => {
        let data = [];

        for (let pi = 0; pi < points.length; pi++) {
          const point = points[pi];

          let offset = 0;
          for (let i = index+1; i < point.length; i++) {
            offset += point[i];
          }

          const datum = [point[0], // date
                         offset,
                         point[index]];
          data.push(datum);
        }

        this.svg.append("g")
            .selectAll(`rect.ev-rect.ev-${name}`)
            .data(data)
            .enter()
            .append("rect")
            .attr("class", `ev-rect ${className}`)
            .attr("x", d => this.xScale(d[0]) + barPadding)
            .attr("y", d => this.yScale(d[2]) - (this.height - this.yScale(d[1])))
            .attr("width", barWidth)
            .attr("height", d => this.height - this.yScale(d[2]))
            .on("mouseover", onMouseover)
            .on("mouseout", onMouseout)
            .append("title")
            .text(p => ""+p[2]);
      }

      renderGroup("successful", "is-success", 2);
      renderGroup("aborted", "is-warning", 3);
      renderGroup("failed", "is-danger", 4);
    },
  });

  graph.update();
}

function evSetupJobRunningTimeMetrics(jobId) {
  const element = document.getElementById("running-time-metrics");

  // Graph
  const graph = new EvTimeGraph(element, {
    yAxisTitle: "Running time",

    legend: [
      {class: "ev-p99", label: "0.99"},
      {class: "ev-p80", label: "0.80"},
      {class: "ev-p50", label: "0.50"},
    ],

    dataURI: function() {
      let granularity = "hour";
      if (this.timeRange == "30d") {
        granularity = "day";
      }

      const query = new URLSearchParams([
        ["start", evDateToUnix(this.startDate)],
        ["end", evDateToUnix(this.endDate)],
        ["granularity", granularity],
      ]);

      return `/jobs/id/${jobId}/metrics/running_times?${query}`;
    },

    updateYScale: function (points) {
      if (!points) { points = []; }

      const maxDurations = points.map(p => d3.max([p[1], p[2], p[3]]));
      const maxDuration = (maxDurations.length>0) ? d3.max(maxDurations) : 3600;

      const maxY = Math.ceil(maxDuration);

      this.yTicks = d3.range(0, maxY, Math.ceil(maxY/6)).concat(maxY);
      this.yTickFormat = (d, i) => { return evFormatShortDuration(d) };
      this.yScale = d3.scaleLinear()
                      .domain([0, maxY])
                      .range([this.height, 0]);
    },

    renderData: function (points) {
      if (!points) { points = []; }

      points.forEach((p) => p[0] = new Date(p[0]*1000));
      points.sort((a, b) => a[0] - b[0]);

      // Tooltips
      onMouseover = function (event, p) {
        d3.select(this)
          .classed("is-active", true)
          .attr("r", 5);
      };

      onMouseout = function (event, p) {
        d3.select(this)
          .classed("is-active", false)
          .attr("r", 2.5);
      };

      // Lines
      renderLine = (name, index) => {
        this.svg.append("path")
            .datum(points)
            .attr("class", `ev-line ev-${name}`)
            .attr("d", d3.line()
                         .x(p => this.xScale(p[0]))
                         .y(p => this.yScale(p[index])));

        this.svg.selectAll(`circle.ev-point.ev-${name}`)
            .data(points)
            .enter()
            .append("circle")
            .attr("class", `ev-point ev-${name}`)
            .attr("cx", p => this.xScale(p[0]))
            .attr("cy", p => this.yScale(p[index]))
            .attr("r", 2.5)
            .on("mouseover", onMouseover)
            .on("mouseout", onMouseout)
            .append("title")
            .text(p => evFormatShortDuration(p[index]));
      };

      renderLine("p99", 1);
      renderLine("p80", 2);
      renderLine("p50", 3);
    },
  });

  graph.update();
}
