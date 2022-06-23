"use strict";

evOnPageLoaded("events", evSetupAutoRefresh);
evOnPageLoaded("events", evSetupEventList);
evOnPageLoaded("event_view", evSetupAutoRefresh);
evOnPageLoaded("event_view", evSetupEventView);

function evSetupEventList() {
  const replayLinkSelector = "#ev-events a[data-action='replay']:not(.ev-disabled)";
  const replayLinks = document.querySelectorAll(replayLinkSelector);
  replayLinks.forEach(link => {
    link.onclick = evOnReplayEventClicked;
  });
}

function evSetupEventView() {
  const replayButton = document.querySelector("button[name='replay']");
  if (replayButton) {
    replayButton.onclick = evOnReplayEventClicked;
  }
}

function evOnReplayEventClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;

  const modal = document.querySelector("#ev-replay-event-modal");

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const replayButton = modal.querySelector("button[name='replay']");
  replayButton.onclick = function () {
    replayButton.classList.add("is-loading");

    const uri = `/events/id/${id}/replay`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        window.location.href = response.data.location;
      })
      .catch (e => {
        evShowError(`cannot replay event: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        replayButton.classList.remove("is-loading");
        evCloseModals();
      });
  };

  evOpenModal(modal);
}
