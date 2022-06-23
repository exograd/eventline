"use strict";

evOnPageLoaded("account_api_keys", evSetupAPIKeys);
evOnPageLoaded("account_api_key_creation", evSetupAPIKeyCreation);
evOnPageLoaded("account_view", evSetupAccountView);

function evSetupAPIKeys() {
  const deleteSelector = "#ev-api-keys a[data-action='delete']";
  const deleteLinks = document.querySelectorAll(deleteSelector);
  deleteLinks.forEach(link => {
    link.onclick = evOnDeleteAPIKeyClicked;
  });
}

function evSetupAPIKeyCreation() {
  const form = document.getElementById("ev-api-key-creation-form");

  onResponse = function(response) {
    const name = response.data["api_key_name"];
    const key = response.data["key"];

    const modal = document.getElementById("ev-api-key-created-modal");
    console.log("modal", modal)

    modal.querySelector(".ev-api-key-name").textContent = name;
    modal.querySelector(".ev-api-key-value").textContent = key;

    const okButton = modal.querySelector("button[name='ok']");
    okButton.onclick = function () {
      window.location = response.data.location;
    };

    evOpenModal(modal);
  };

  evSetupForm(form, {onResponse: onResponse});
}

function evOnDeleteAPIKeyClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;
  const name = link.dataset.name;

  const modal = document.querySelector("#ev-delete-api-key-modal");

  modal.querySelector(".ev-api-key-name").textContent = name;

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const deleteButton = modal.querySelector("button[name='delete']");
  deleteButton.onclick = function () {
    deleteButton.classList.add("is-loading");

    const uri = `/account/api_keys/id/${id}/delete`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        location.reload();
      })
      .catch (e => {
        evShowError(`cannot delete API key ${name}: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        deleteButton.classList.remove("is-loading");
      });
  }

  evOpenModal(modal);
}

function evSetupAccountView() {
}
