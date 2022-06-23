"use strict";

evOnPageLoaded("identities", evSetupIdentities);
evOnPageLoaded("identities", evSetupAutoRefresh);
evOnPageLoaded("identity_view", evSetupIdentityView);
evOnPageLoaded("identity_configuration", evSetupIdentityForm);
evOnPageLoaded("identity_logs", evSetupAutoRefresh);

function evSetupIdentities() {
  const identities = document.getElementById("ev-identities");

  const refreshSelector = "a[data-action='refresh']:not(.ev-disabled)";
  const refreshLinks = identities.querySelectorAll(refreshSelector);
  refreshLinks.forEach(link => {
    link.onclick = evOnRefreshIdentityClicked;
  });

  const deleteSelector = "a[data-action='delete']:not(.ev-disabled)";
  const deleteLinks = identities.querySelectorAll(deleteSelector);
  deleteLinks.forEach(link => {
    link.onclick = evOnDeleteIdentityClicked;
  });
}

function evSetupIdentityView() {
  const deleteButton = document.querySelector("button[name='delete']");
  deleteButton.onclick = evOnDeleteIdentityClicked;

  const refreshButton = document.querySelector("button[name='refresh']");
  refreshButton.onclick = evOnRefreshIdentityClicked;
}

function evOnRefreshIdentityClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;
  const name = link.dataset.name;

  const modal = document.querySelector("#ev-refresh-identity-modal");

  modal.querySelector(".ev-identity-name").textContent = name;

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const refreshButton = modal.querySelector("button[name='refresh']");
  refreshButton.onclick = function () {
    refreshButton.classList.add("is-loading");

    const uri = `/identities/id/${id}/refresh`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        window.location.reload();
      })
      .catch (e => {
        evShowError(`cannot refresh identity ${name}: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        refreshButton.classList.remove("is-loading");
      });
  }

  evOpenModal(modal);
}

function evOnDeleteIdentityClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;
  const name = link.dataset.name;

  const modal = document.querySelector("#ev-delete-identity-modal");

  modal.querySelector(".ev-identity-name").textContent = name;

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const deleteButton = modal.querySelector("button[name='delete']");
  deleteButton.onclick = function () {
    deleteButton.classList.add("is-loading");

    const uri = `/identities/id/${id}/delete`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        window.location.href = response.data.location;
      })
      .catch (e => {
        evShowError(`cannot delete identity ${name}: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        deleteButton.classList.remove("is-loading");
      });
  }

  evOpenModal(modal);
}

function evSetupIdentityForm() {
  const form = document.getElementById("ev-identity-form");

  // Identity data must always be present even if they are just an empty
  // object.
  evSetupForm(form, {baseData: {data: {}}});

  const connectorSelect = document.getElementById("ev-connector-select");
  const typeSelect = document.getElementById("ev-type-select");

  connectorSelect.onchange = ((event) => {
    if (typeSelect) {
      typeSelect.value = null;
    }
    evReloadIdentityFormTypes(true);
  });

  if (typeSelect) {
    typeSelect.onchange = evReloadIdentityFormData;
  }

  const identityId = form.dataset.identityId;
  var currentType;

  if (identityId) {
    connectorSelect.value = form.dataset.connector;
    currentType = form.dataset.type;
  } else {
    connectorSelect.value = "generic";
  }

  // Do not reload form data if we are editing an identity: data fields were
  // populated in the template, we do not want to erase them.
  const reloadFormData = (identityId == null)
  evReloadIdentityFormTypes(reloadFormData, currentType);
}

function evReloadIdentityFormTypes(reloadFormData, currentType = null) {
  const form = document.getElementById("ev-identity-form");
  const connectorSelect = document.getElementById("ev-connector-select");
  const typeSelectContainer = document.getElementById("ev-type-select-container");
  var typeSelect = document.getElementById("ev-type-select");
  const submitButtons = form.querySelectorAll("button[type='submit']");

  const connector = connectorSelect.value;

  const query = new URLSearchParams();
  if (currentType) {
    query.append("current_type", currentType)
  }

  const uri = "/identities/connector/" + encodeURI(connector) + "/types" +
        "?" + query;

  const request =  {
    method: "GET",
    headers: {
      "Accept": "text/html",
    },
  };

  connectorSelect.disabled = true;
  submitButtons.forEach((button) => {button.disabled = true});
  if (typeSelect) {
    typeSelect.disabled = true;
  }

  evFetch(uri, request, decodeFunc = null)
    .then(response => {
      typeSelectContainer.innerHTML = response.data;

      typeSelect = document.getElementById("ev-type-select");
      typeSelect.onchange = evReloadIdentityFormData;

      if (!currentType) {
        const option = typeSelect.querySelector("option");
        if (option) {
          const optionValue = typeSelect.querySelector("option").value;
          typeSelect.value = optionValue;
        }
      }

      if (reloadFormData) {
        evReloadIdentityFormData();
      }
    })
    .catch(e => {
      evShowError(`cannot load identity types: ${e.message}`);
    })
    .finally(() => {
      connectorSelect.disabled = false;
      submitButtons.forEach((button) => {button.disabled = false});
      typeSelect.disabled = false;
    });
}

function evReloadIdentityFormData() {
  const form = document.getElementById("ev-identity-form");
  const connectorSelect = document.getElementById("ev-connector-select");
  const typeSelect = document.getElementById("ev-type-select");
  const submitButtons = form.querySelectorAll("button[type='submit']");
  const dataSection = document.getElementById("ev-identity-data");

  const connector = connectorSelect.value;
  const type = typeSelect.value;

  const uri = "/identities/connector/" + encodeURI(connector) +
        "/type/" + encodeURI(type) + "/data";
  const request =  {
    method: "GET",
    headers: {
      "Accept": "text/html",
    },
  };

  connectorSelect.disabled = true;
  submitButtons.forEach((button) => {button.disabled = true});
  typeSelect.disabled = true;

  evFetch(uri, request, decodeFunc = null)
    .then(response => {
      dataSection.innerHTML = response.data;
      evCreateFormHelpElements(dataSection);
      evSetupMultiSelects();
    })
    .catch(e => {
      evShowError(`cannot load identity form data: ${e.message}`);
    })
    .finally(() => {
      connectorSelect.disabled = false;
      submitButtons.forEach((button) => {button.disabled = false});
      typeSelect.disabled = false;
    });
}
