"use strict";

evOnPageLoaded("%any", evSetupProjectDialog);
evOnPageLoaded("projects", evSetupProjects);

function evSetupProjectDialog() {
  const link = document.getElementById("project-dialog-link");
  if (link) {
    link.onclick = evOnProjectDialogLinkClicked
  }
}

function evOnProjectDialogLinkClicked(event) {
  event.preventDefault();

  const link = event.target;

  if (link.dataset.loading == "true") {
    return;
  }

  link.dataset.loading = "true";

  const uri = `/projects/dialog`
  const request = {
    method: "GET",
    headers: {
      "Accept": "text/html",
    },
  };

  evFetch(uri, request, decodeFunc = null)
      .then(response => {
        const body = document.getElementsByTagName("body")[0]
        body.insertAdjacentHTML("beforeend", response.data);

        const modal = document.getElementById("ev-project-dialog-modal");
        evInitProjectDialog(modal);
      })
      .catch (e => {
        evShowError(`cannot load project selector: ${e.message}`);
      })
      .finally(() => {
        delete link.dataset["loading"]
      });
}

function evInitProjectDialog(modal) {
  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const selectLinkSelector = "#ev-projects a.ev-project-selector";
  const selectLinks = document.querySelectorAll(selectLinkSelector);
  selectLinks.forEach(link => {
    link.onclick = (ev) => evOnSelectProjectClicked(ev, false);
  });

  evOpenModal(modal);
}

function evSetupProjects() {
  const selectLinkSelector = "#ev-projects a.ev-project-selector";
  const selectLinks = document.querySelectorAll(selectLinkSelector);
  selectLinks.forEach(link => {
    link.onclick = (ev) => evOnSelectProjectClicked(ev, true);
  });

  const deleteButtonSelector = "#ev-projects a[data-action='delete']";
  const deleteButtons = document.querySelectorAll(deleteButtonSelector);
  deleteButtons.forEach(button => {
    button.onclick = evOnDeleteProjectClicked;
  });
}

function evOnSelectProjectClicked(event, redirect) {
  event.preventDefault();

  const button = event.target;
  const id = button.dataset.id;
  const name = button.dataset.name;

  button.classList.add("is-loading");

  const uri = `/projects/id/${id}/select`
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
      .then(response => {
        if (redirect) {
          window.location.href = response.data.location;
        } else {
          const currentURI = new URL(window.location);
          const currentQuery = currentURI.searchParams;

          // Drop any before/after parameter since the object ids will not
          // exist in the new project.
          currentQuery.delete("before");
          currentQuery.delete("after");

          window.location.href = currentURI;
        }
      })
      .catch (e => {
        evShowError(`cannot select project ${name}: ${e.message}`);
      })
      .finally(() => {
        button.classList.remove("is-loading");
      });
}

function evOnDeleteProjectClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;
  const name = link.dataset.name;

  const modal = document.querySelector("#ev-delete-project-modal");

  modal.querySelector(".ev-project-name").textContent = name;

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const deleteButton = modal.querySelector("button[name='delete']");
  deleteButton.onclick = function () {
    deleteButton.classList.add("is-loading");

    const uri = `/projects/id/${id}/delete`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        location.reload();
      })
      .catch (e => {
        evShowError(`cannot delete project ${name}: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        deleteButton.classList.remove("is-loading");
        evCloseModals();
      });
  };

  evOpenModal(modal);
}
