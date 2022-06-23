"use strict";

function evOpenModal(element) {
  const rootElement = document.documentElement;
  rootElement.classList.add("is-clipped");

  element.classList.add("is-active");
}

function evCloseModals() {
  const rootElement = document.documentElement;
  rootElement.classList.remove("is-clipped");

  const modals = document.querySelectorAll(".modal");
  modals.forEach(modal => {
    modal.classList.remove("is-active");
  });
}

function evIsModalActive() {
  return document.querySelectorAll(".modal.is-active").length > 0
}
