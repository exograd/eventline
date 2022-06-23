"use strict";

class EvAPIError extends Error {
  constructor(status, code, data, ...args) {
    super(...args);

    this.status = status;
    this.code = code;
    this.data = data;
  }
}

class EvMiscError extends Error {
  constructor(status, ...args) {
    super(...args);

    this.status = status;
  }
}

function evFetch(uri, request, decodeFunc = JSON.parse, redirect = false) {
  defaultHeaders = {
    "Accept": "application/json",
    "Content-Type": "application/json"
  }

  request.headers ?? (request.headers = {});
  for (const name in defaultHeaders) {
    const value = defaultHeaders[name];
    request.headers[name] ?? (request.headers[name] = value);
  }

  request.credentials ?? (request.credentials = "same-origin");

  return new Promise((resolve, reject) => {
    fetch(uri, request)
      .then(response => {
        if (response.redirected && redirect) {
          window.location.href = response.url
          resolve({status: response.status, redirected: true,
                   url: response.url});
          return;
        }

        response.text()
          .then(text => {
            if (response.ok) {
              if (decodeFunc === null) {
                decodeFunc = ((data) => {return data});
              }
            } else {
              decodeFunc = JSON.parse;
            }

            const status = response.status;

            let data = null;

            if (text !== "") {
              try {
                data = decodeFunc(text)
              } catch (e) {
                let msg;
                if (response.ok) {
                  msg = `invalid response data: ${e}`;
                } else {
                  msg = `request failed with status ${status}`;
                  if (response.statusText) {
                    msg += `: ${response.StatusText}`;
                  }
                }

                reject(new EvMiscError(status, msg));
                return;
              }
            }

            if (response.ok) {
              resolve({status: response.status, data: data});
            } else {
              const code = data.code ?? "unknown_error";
              const message = data.error;
              const errorData = data.data ?? {};

              reject(new EvAPIError(status, code, errorData, message));
            }
          })
      })
      .catch (e => {
        console.error("fetch error: ", e);
        reject(new Error("network error"));
      })
  });
}
