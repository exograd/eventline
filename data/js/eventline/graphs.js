"use strict";

class EvGraph {
  constructor(element, options = {}) {
    this.element = element;

    const defaultOptions = {
      xAxisTitle: "",
      yAxisTitle: "",

      legend: [],

      dataURI: undefined,

      updateXScale: undefined,
      updateYScale: undefined,
      renderData: undefined,
    };

    this.options = {...defaultOptions, ...options};

    this.margins = {top: 30, right: 20, bottom: 45, left: 60};

    const svg = this.element.querySelector("svg");

    this.width = svg.clientWidth - this.margins.left - this.margins.right;
    this.height = svg.clientHeight - this.margins.top - this.margins.bottom;

    this.svg = d3.select(svg)
                 .append("g")
                 .attr("transform",
                       `translate(${this.margins.left}, ${this.margins.top})`);

    if (!window.evGraphs) {
      window.evGraphs = []
    }
    window.evGraphs.push(this)
  }

  pre_update() {
  }

  sync_with_others() {
  }

  update() {
    this.pre_update();

    let uri;
    if (typeof this.options.dataURI === "string") {
      uri = this.options.dataURI;
    } else if (typeof this.options.dataURI === "function") {
      uri = this.options.dataURI.call(this);
    }

    const request = {
      method: "GET",
    };

    evFetch(uri, request)
      .then(response => {
        this.render(response.data);
      })
      .catch (e => {
        evShowError(`cannot fetch metric data: ${e.message}`);
      });
  }

  clear() {
    this.svg.selectAll("*").remove();
  }

  render(data) {
    this.clear();

    this.options.updateXScale.call(this, data);
    this.options.updateYScale.call(this, data);

    this.renderXAxis();
    this.renderYAxis();
    this.renderGrid();

    this.options.renderData.call(this, data);

    if (this.options.legend.length > 0) {
      this.renderLegend();
    }
  }

  renderXAxis() {
    this.svg.append("g")
        .attr("transform", `translate(0, ${this.height})`)
        .call(d3.axisBottom(this.xScale)
                .tickValues(this.xTicks)
                .tickFormat(this.xTickFormat)
                .tickSizeOuter(0));

    this.svg.append("text")
        .attr("class", "ev-axis-title")
        .attr("text-anchor", "middle")
        .attr("transform",
              `translate(${this.width/2}, ${this.height+this.margins.bottom-4})`)
        .text(this.options.xAxisTitle);
  }

  renderYAxis() {
    this.svg.append("g")
        .call(d3.axisLeft(this.yScale)
                .tickValues(this.yTicks)
                .tickFormat(this.yTickFormat)
                .tickSizeOuter(0));

    this.svg.append("text")
        .attr("class", "ev-axis-title")
        .attr("transform",
              `translate(${-this.margins.left+4}, ${-this.margins.top+16})`)
        .text(this.options.yAxisTitle);
  }

  renderGrid() {
    this.svg
        .append("g")
        .attr("class", "ev-grid")
        .selectAll("line")
        .data(this.xTicks)
        .join("line")
        .attr("x1", x => this.xScale(x))
        .attr("x2", x => this.xScale(x))
        .attr("y1", 0)
        .attr("y2", this.height);

    this.svg
        .append("g")
        .attr("class", "ev-grid")
        .selectAll("line")
        .data(this.yTicks)
        .join("line")
        .attr("x1", 0)
        .attr("x2", this.width)
        .attr("y1", y => this.yScale(y))
        .attr("y2", y => this.yScale(y));
  }

  renderLegend() {
    const leftMargin = 4;

    const radius = 5;

    const entryPadding = 4;
    const entryHeight = 16 + entryPadding*2;

    const entryOffset = leftMargin + entryPadding*2;
    const entryTextOffset = entryOffset + radius + entryPadding*2;

    let width = 160;
    const height = this.options.legend.length * entryHeight;

    const legend = this.svg
                       .append("g")
                       .classed("ev-legend", true);

    legend.append("rect")
                  .classed("ev-background", true)
                  .attr("x", leftMargin)
                  .attr("y", 0)
                  .attr("width", width)
                  .attr("height", height);

    legend.append("rect")
                  .classed("ev-border", true)
                  .attr("x", 4)
                  .attr("y", 0)
                  .attr("width", width)
                  .attr("height", height)
                  .attr("rx", 2) // CSS $radius-small
                  .attr("ry", 2);

    legend.selectAll("circle.ev-point")
      .data(this.options.legend)
      .enter()
      .append("circle")
      .attr("class", e => e.class)
      .classed("ev-point", true)
      .attr("cx", e => entryOffset + radius/2)
      .attr("cy", (e, i) => i*entryHeight + entryHeight/2)
      .attr("r", radius);

    legend.selectAll("text")
      .data(this.options.legend)
      .enter()
      .append("text")
      .attr("x", e => entryTextOffset)
      .attr("y", (e, i) => i*entryHeight + entryHeight/2)
      .text(e => e.label);

    const texts = Array.from(legend.node().querySelectorAll("text"));
    const maxX = d3.max(texts.map((e) => {
      const bbox = e.getBBox();
      return bbox.x + bbox.width;
    }));

    width = maxX - leftMargin + entryPadding;

    legend.selectAll("rect.ev-background, rect.ev-border")
          .attr("width", width);
  }
}

class EvTimeGraph extends EvGraph {
  constructor(element, options = {}) {
    super(element, options);

    // Time range select
    this.timeRangeSelect = this.element.querySelector("select[name='time-range']");
    this.timeRangeSelect.onchange = (e) => {
      this.update();
      this.sync_with_others();
    }

    this.options.updateXScale = function (points) {
      if (!points) { points = []; }

      const interval = (() => {
        switch (this.timeRange) {
        case "30d": return d3.utcDay;
        case "7d": return d3.utcDay;
        case "24h": return d3.utcHour;
        }
      })();

      this.xTicks = interval.range(this.startDate, this.endDate)
                            .concat(this.endDate);

      this.xTickFormat = (() => {
        const formatDay = d3.utcFormat("%m/%d");
        const formatHour = d3.utcFormat("%Hh");

        switch (this.timeRange) {
        case "30d":
          return (d, i) => { return (i%5 == 0) ? formatDay(d) : "" };
        case "7d":
          return (d, i) => { return formatDay(d) };
        case "24h":
          return (d, i) => { return formatHour(d) };
        }
      })();

      this.xScale = d3.scaleTime()
                      .domain([this.startDate, this.endDate])
                      .range([0, this.width]);
    };

    this.options.xAxisTitle = "Date (UTC)";
  }

  pre_update() {
    super.pre_update();

    const now = new Date();

    this.timeRange = this.timeRangeSelect.value;

    if (this.timeRange == "30d") {
      this.startDate = evNextDayStart(new Date(now - 30*86400*1000));
      this.endDate = evNextDayStart(now);
    } else if (this.timeRange == "7d") {
      this.startDate = evNextDayStart(new Date(now - 7*86400*1000));
      this.endDate = evNextDayStart(now);
    } else {
      this.startDate = evNextHourStart(new Date(now - 86400*1000));
      this.endDate = evNextHourStart(now);
    }
  }

  sync_with_others() {
    const currentGraph = this;
    const currentTimeRange = this.timeRange;

    window.evGraphs.forEach((graph) => {
      if (graph == currentGraph) {
        return;
      }

      if (graph.timeRange && graph.timeRange != currentTimeRange) {
        graph.timeRangeSelect.value = currentTimeRange;
        graph.timeRange = currentTimeRange;
        graph.update()
      }
    })
  }
}
