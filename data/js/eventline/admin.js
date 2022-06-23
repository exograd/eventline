"use strict";

evOnPageLoaded("admin_accounts", evSetupAdminAccounts);

function evSetupAdminAccounts() {
  const deleteLinkSelector = "#ev-accounts a[data-action='delete']";
  const deleteLinks = document.querySelectorAll(deleteLinkSelector);
  deleteLinks.forEach(link => {
    link.onclick = evOnDeleteAccountClicked;
  });
}

function evOnDeleteAccountClicked(event) {
  event.preventDefault();

  const link = event.target;
  const id = link.dataset.id;
  const username = link.dataset.username;

  const modal = document.querySelector("#ev-delete-account-modal");

  modal.querySelector(".ev-account-username").textContent = username;

  modal.querySelectorAll("button[name='cancel']").forEach(button => {
    button.onclick = evCloseModals;
  });

  const deleteButton = modal.querySelector("button[name='delete']");
  deleteButton.onclick = function () {
    deleteButton.classList.add("is-loading");

    const uri = `/admin/accounts/id/${id}/delete`
    const request = {
      method: "POST"
    };

    evFetch(uri, request)
      .then(response => {
        location.reload();
      })
      .catch (e => {
        evShowError(`cannot delete account ${username}: ${e.message}`);
        evCloseModals();
      })
      .finally(() => {
        deleteButton.classList.remove("is-loading");
        evCloseModals();
      });
  };

  evOpenModal(modal);
}
