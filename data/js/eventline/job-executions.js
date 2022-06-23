"use strict";

evOnPageLoaded("job_execution_view", evSetupJobExecutionView);
evOnPageLoaded("job_execution", evSetupJobExecution);

function evSetupJobExecutionView() {
  window.evStepStates = new Map();

  const container = document.getElementById("ev-content-container");
  const jeId = container.dataset.jobExecutionId;

  evUpdateJobExecutionView(jeId);
}

function evUpdateJobExecutionView(jeId) {
  const uri = `/job_executions/id/${jeId}/content`
  const request = {
    method: "GET"
  };

  evFetch(uri, request, decodeFunc = null)
    .then(response => {
      evRenderJobExecutionView(response.data);
    })
    .catch (e => {
      evShowError(`cannot fetch content: ${e.message}`);
    })
    .finally(() => {
      const je = document.getElementById("ev-job-execution");
      const jeStatus = je.dataset.status;

      window.evAutoFold = (jeStatus == "started");

      if (jeStatus == "created" && window.evPreviousJobExecutionStatus != "created") {
        // The job execution was restarted, let us start from scratch
        window.evStepStates = new Map();
      }
      window.evPreviousJobExecutionStatus = jeStatus;

      let delay = 2500;
      if (['successful', 'aborted', 'failed'].includes(jeStatus)) {
        delay = 15000;
      }

      setTimeout(evUpdateJobExecutionView, delay, jeId);
    });
}

function evRenderJobExecutionView(content) {
  const container = document.getElementById("ev-content-container");
  container.innerHTML = content;

  const abortButton =
        document.querySelector("#ev-job-execution button[name='abort']");
  abortButton.onclick = evOnAbortJobExecutionClicked

  const restartButton =
        document.querySelector("#ev-job-execution button[name='restart']");
  restartButton.onclick = evOnRestartJobExecutionClicked

  evSetupStepFolding();
}

function evSetupStepFolding() {
  const steps = document.getElementById("ev-steps");
  const headers = steps.querySelectorAll(".ev-step-header");

  const jobExecution = document.getElementById("ev-job-execution");
  const jobExecutionStatus = jobExecution.dataset.status;

  let lastRunStep;

  headers.forEach((header) => {
    const step = header.closest(".ev-step");
    const stepId = step.dataset.id;
    const stepStatus = step.dataset.status;

    const body = step.querySelector(".ev-step-body");
    const icon = step.querySelector(".ev-folding-icon i");

    let stepState = window.evStepStates.get(stepId);
    let folded = true;

    if (stepStatus != "created") {
      lastRunStep = step;
    }

    if (stepState) {
      folded = stepState.folded;
    } else {
      if (window.evAutoFold) {
        folded = (stepStatus == "created");

        if (!folded) {
          // Once automatically open, keep it open as long as it does not
          // restart.
          stepState = {folded: false};
          window.evStepStates.set(stepId, stepState);
        }
      } else {
        folded = true;
      }
    }

    if (folded) {
      body.classList.add("is-hidden");
      icon.classList.add("mdi-chevron-right");
      icon.classList.remove("mdi-chevron-down");
    } else {
      body.classList.remove("is-hidden");
      icon.classList.remove("mdi-chevron-right");
      icon.classList.add("mdi-chevron-down");
    }

    const h1 = header.querySelector("h1");
    h1.onclick = (event) => {
      body.classList.toggle("is-hidden");
      icon.classList.remove("mdi-chevron-right", "mdi-chevron-down");

      const hidden = body.classList.contains("is-hidden");
      if (hidden) {
        icon.classList.add("mdi-chevron-right");
      } else {
        icon.classList.add("mdi-chevron-down");
      }

      let stepState = window.evStepStates.get(stepId);
      if (!stepState) {
        stepState = {};
      }

      stepState.folded = hidden;
      window.evStepStates.set(step.dataset.id, stepState);
    };
  });
}

function evSetupJobExecution() {
  const form = document.getElementById("ev-job-execution-form");

  // Command parameters must always be present even if they are just an empty
  // object, which happens when a command has no parameter.
  evSetupForm(form, {baseData: {parameters: {}}});
}

function evOnAbortJobExecutionClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;

  const modal = document.querySelector("#ev-abort-job-modal");

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const abortButton = modal.querySelector("button[name='abort']");
  abortButton.onclick = function () {
    abortButton.classList.add("is-loading");

    const uri = `/job_executions/id/${id}/abort`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        location.reload();
      })
      .catch (e => {
        evShowError(`cannot abort job: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        abortButton.classList.remove("is-loading");
        evCloseModals();
      });
  };

  evOpenModal(modal);
}

function evOnRestartJobExecutionClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;

  const modal = document.querySelector("#ev-restart-job-modal");

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const restartButton = modal.querySelector("button[name='restart']");
  restartButton.onclick = function () {
    restartButton.classList.add("is-loading");

    const uri = `/job_executions/id/${id}/restart`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        location.reload();
      })
      .catch (e => {
        evShowError(`cannot restart job: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        restartButton.classList.remove("is-loading");
        evCloseModals();
      });
  };

  evOpenModal(modal);
}
