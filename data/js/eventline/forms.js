"use strict";

function evSetupForms() {
  document.querySelectorAll("form.ev-auto-form").forEach((form) => {
    evSetupForm(form, {})
  });
}

function evSetupForm(form, options) {
  evCreateFormHelpElements(form);

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    evSubmitForm(form, event.submitter, options);
  });
}

function evCreateFormHelpElements(container) {
  const fields = container.querySelectorAll(".field");

  fields.forEach((field) => {
    let help = document.createElement("p");
    help.classList.add("help", "is-danger");

    field.appendChild(help);
  });
}

function evSubmitForm(form, button, options) {
  if (form.checkValidity() === false) {
	form.reportValidity()
	return
  }

  const submitButtons = form.querySelectorAll("button[type='submit']");
  submitButtons.forEach(evDisableSubmitButton);

  const formData = evCollectFormData(form, button, options);

  const uri = form.action;
  const request = {
    method: "POST",
    body: JSON.stringify(formData)
  };

  evClearNotifications();
  evClearFormErrorAnnotations(form);

  evFetch(uri, request)
    .then(response => {
      if ("onResponse" in options) {
        options.onResponse(response);
      } else if (response.data && response.data.location) {
        window.location = response.data.location;
      } else {
        window.location.reload();
      }
    })
    .catch(e => {
      if ((e instanceof EvAPIError) && e.code == "invalid_request_body") {
        evShowError("Invalid form data.");
        evAnnotateInvalidForm(form, e.data.validation_errors);
      } else {
        evShowError(e.message);
      }
    })
    .finally(() => {
      submitButtons.forEach(evEnableSubmitButton);
    });
}

function evAnnotateInvalidForm(form, validationErrors) {
  evClearFormErrorAnnotations(form);

  validationErrors.forEach(e => {
    const pointer = e.pointer;
    const message = e.message;

    if (pointer === "") {
      console.error(`invalid top-level value error: ${message}`);
      return;
    }

    const input = form.querySelector("[name='" + pointer + "']");
    if (input === null) {
      console.error(`no input element found for pointer ${pointer}`);
      return;
    }

    const control = input.closest(".control");
    const field = control.closest(".field");
    const help = field.querySelector("p.help.is-danger");

    var dangerElement;
    if (input.tagName == "SELECT") {
      dangerElement = input.parentNode; // the div.select wrapper
    } else {
      dangerElement = input;
    }
    dangerElement.classList.add("is-danger");

    help.textContent = evSentence(message);
  });
}

function evClearFormErrorAnnotations(form) {
  const helps = form.querySelectorAll("p.help.is-danger");
  helps.forEach(help => {
    help.textContent = "";
  });

  const inputSelector = "div.control input, div.control select";
  const inputs = form.querySelectorAll(inputSelector);
  inputs.forEach(input => {
    input.classList.remove("is-danger");
  });
}

function evCollectFormData(form, button, options) {
  const elements = form.elements;

  let data = options.baseData ?? {};

  for (let i = 0; elements[i] !== undefined; i++) {
    const input = elements[i];

    if (input.nodeName === "BUTTON" && input.type === "submit") {
      if (input !== button) {
        continue;
      }

      if (!input.name || input.name.length == 0 || input.name[0] != '/') {
        continue;
      }
    }

    if (input.nodeName === "INPUT" && input.type === "radio") {
      if (!input.checked) {
        continue;
      }
    }

    const name = input.name;
    if (name === "") {
      continue;
    }

    const value = (() => {
      switch (input.type) {
      case "button":
        return input.value;
      case "checkbox":
        return input.checked;
      case "radio":
        if (input.value == "true") {
          return true;
        } else if (input.value == "false") {
          return false;
        }
        return input.value;
      case "select-multiple":
        return [...input.options].filter(o => o.selected).map(o => o.value);
      case 'number':
        if (input.value == "") {
          return input.value;
        } else {
          const numberValue = JSON.parse(input.value);
          if (typeof numberValue == "number") {
            return numberValue;
          } else {
            // Let server validation return a nice error
            return input.value;
          }
        }
      default:
        if (input.classList.contains("ev-list-input")) {
          return input.value
                      .split(",")
                      .map(v => v.trim())
                      .filter(v => v.length > 0);
        } else {
          return input.value;
        }
      }
    })();

    if (input.type != "hidden") {
      const control = input.closest(".control");
      const field = control.closest(".field");

      if (!field.classList.contains("ev-required") && value === "") {
        continue;
      }
    }

    const pointer = evParseJSONPointer(name);
    if (pointer.length === 0) {
      throw new Error("invalid empty json pointer for form field '"
                      + name + "'");
    }

    evFormDataInsert(data, pointer, value);
  }

  return data;
}

function evFormDataInsert(data, pointer, value) {
  const key = pointer[0];

  if (pointer.length == 1) {
      data[key] = value;
  } else {
    if (!(key in data)) {
      data[key] = {};
    }

    evFormDataInsert(data[key], pointer.slice(1), value);
  }
}

function evDisableSubmitButton(button) {
  button.disabled = true;
  button.classList.add("is-loading");
}

function evEnableSubmitButton(button) {
  button.classList.remove("is-loading");
  button.disabled = false;
}
