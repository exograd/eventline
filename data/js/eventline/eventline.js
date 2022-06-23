"use strict";

evOnPageLoaded("%any", evSetupMainMenu);
evOnPageLoaded("%any", evSetupSecrets);
evOnPageLoaded("%any", evSetupTagDropdowns);
evOnPageLoaded("%any", evSetupForms);
evOnPageLoaded("%any", evSetupLogout);
evOnPageLoaded("%any", evSetupHighlightJS);
evOnPageLoaded("error", evSetupErrorGoBack);

function evSetupSecrets() {
  const elements = document.querySelectorAll(".ev-secret");

  elements.forEach(evSetupSecret);
}

function evSetupSecret(element) {
  element.dataset.evSecretText = element.textContent;

  element.onclick = function (e) {
    if (element.classList.contains("is-active")) {
      evHideSecret(element);
    } else {
      evShowSecret(element);
    }

    element.classList.toggle("is-active");
  };

  evHideSecret(element);
}

function evShowSecret(element) {
  element.textContent = element.dataset.evSecretText;
  element.title = "Click to hide secret";
}

function evHideSecret(element) {
  let tags = document.createElement("div");
  tags.classList.add("tags", "has-addons");
  tags.title = "Click to show secret";

  let iconSpan = document.createElement("span");
  iconSpan.classList.add("tag", "is-success");
  let icon = document.createElement("i");
  icon.classList.add("mdi", "mdi-18px", "mdi-lock-outline");
  iconSpan.appendChild(icon);

  let labelSpan = document.createElement("span");
  labelSpan.classList.add("tag");
  let label = document.createElement("span");
  label.appendChild(document.createTextNode("secret"));
  labelSpan.appendChild(label);

  tags.appendChild(iconSpan);
  tags.appendChild(labelSpan);

  element.innerHTML = "";
  element.appendChild(tags);
  element.removeAttribute("title");
}

function evSetupTagDropdowns() {
  window.evDropdownClickListener = function() {
    document.querySelectorAll(".dropdown").forEach((dropdown) => {
      dropdown.classList.remove("is-active");
    });

    return true;
  }

  document.removeEventListener("click", window.evDropdownClickListener);
  document.addEventListener("click", window.evDropdownClickListener);

  const tags = document.querySelectorAll(".dropdown-trigger .tag");

  tags.forEach((tag) => {
    const dropdown = tag.closest(".dropdown");
    const menu = dropdown.querySelector(".dropdown-menu");

    tag.addEventListener("click", (ev) => {
      ev.stopImmediatePropagation();

      document.querySelectorAll(".dropdown").forEach((otherDropdown) => {
        otherDropdown.classList.remove("is-active");
      });

      dropdown.classList.add("is-active");
    });
  });
}

function evIsDropdownActive() {
  return document.querySelectorAll(".dropdown.is-active").length > 0
}

function evSetupLogout() {
  if (evIsLoggedIn()) {
    const link = document.querySelector(".menu li a[data-id='logout']");
    if (link) {
      link.onclick = (event) => {
        event.preventDefault();
        evLogout();
      };
    }
  }
}

function evLogout() {
  const uri = "/logout";
  const request = {
    method: "POST"
  };

  evFetch(uri, request)
    .then(response => {
      window.location.href = response.data.location;
    })
    .catch (e => {
      evShowError(`cannot log out: ${e.message}`);
    });
}

function evSetupHighlightJS() {
  hljs.highlightAll();
}

function evSetupErrorGoBack() {
  const buttons = document.querySelectorAll(".button.ev-go-back");
  buttons.forEach(button => {
    button.onclick = function (e) {
      e.preventDefault();
      javascript:history.back();
    }
  });
}

function evPageId() {
  const html = document.getElementsByTagName("html")[0];
  return html.dataset.pageId;
}

function evIsLoggedIn() {
  const html = document.getElementsByTagName("html")[0];
  return html.dataset.isLoggedIn == "true";
}

function evOnPageLoaded(id, fun) {
  if (!window.evPageLoadFunctions) {
    window.evPageLoadFunctions = [];
  }

  document.addEventListener("DOMContentLoaded", (event) => {
    const pageId = evPageId();

    let enableFun = false;
    if (typeof id === "string") {
      enableFun = (id == "%any" || id == pageId);
    } else if (typeof id === "function") {
      enableFun = !!id();
    }

    if (enableFun) {
      window.evPageLoadFunctions.push(fun);
      fun();
    }
  });
}

function evSetupAutoRefresh() {
  if (window.evAutoRefreshTimer) {
    return;
  }

  evScheduleAutoRefresh();
}

function evDisableAutoRefresh() {
  clearTimeout(window.evAutoRefreshTimer);
  delete(window.evAutoRefreshTimer);
}

function evScheduleAutoRefresh() {
  const timer = setTimeout(evAutoRefresh, 5000)
  window.evAutoRefreshTimer = timer;
}

function evAutoRefresh() {
  const uri = window.location.href;
  const request = {
    method: "GET"
  };

  // We use redirect=true so that we actually change the current page on a
  // redirect response. This is especially important for authentication
  // errors: if our session expires, we want to be redirect to the login page,
  // instead of staying on the same page with the content of the login page.
  evFetch(uri, request, decodeFunc = null, redirect = true)
    .then(response => {
      if (response.redirected) {
        return;
      }

      if (evIsDropdownActive()) {
        return;
      }

      if (evIsModalActive()) {
        return;
      }

      const html = document.createElement("html");
      html.innerHTML = response.data;

      const oldContent = document.getElementById("ev-content");
      const newContent = html.querySelector("#ev-content");

      const oldNotifications = oldContent.querySelector("#ev-notifications");
      const newNotifications = newContent.querySelector("#ev-notifications");
      newNotifications.parentNode.replaceChild(oldNotifications, newNotifications);

      oldContent.parentNode.replaceChild(newContent, oldContent);

      if (window.evPageLoadFunctions) {
        window.evPageLoadFunctions.forEach((fun) => {
          fun();
        });
      }
    })
    .catch (e => {
      console.error(`cannot fetch ${uri}: ${e}`)
    })
    .finally(() => {
      evScheduleAutoRefresh();
    });
}

function evIsPrivatePage() {
  const html = document.getElementsByTagName("html")[0];
  return html.dataset.isPublicPage != "true";
}
