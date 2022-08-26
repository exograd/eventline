document.addEventListener("DOMContentLoaded", event => {
  const titles = document.querySelectorAll(".sectlevel1 > li > a");
  titles.forEach(evSetupTOCTitle);
});

function evSetupTOCTitle(title) {
  const parent = title.parentNode;

  const mdi = document.createElement("i");
  mdi.classList.add("mdi");

  const icon = document.createElement("span");
  icon.classList.add("icon");
  icon.appendChild(mdi);

  title.prepend(icon);

  title.onclick = evOnTOCTitleClick;
  evFoldTOCTitle(title);
}

function evOnTOCTitleClick(ev) {
  const title = ev.target.closest("a");

  const titles = document.querySelectorAll(".sectlevel1 > li > a");
  titles.forEach(otherTitle => {
    if (otherTitle != title) {
      evFoldTOCTitle(otherTitle);
    }
  });

  const isFolded = (title.dataset.folded == "true");
  if (isFolded) {
    evExpandTOCTitle(title);
  } else {
    evFoldTOCTitle(title);
  }
}

function evFoldTOCTitle(title) {
  const subtitleList = title.nextElementSibling;
  subtitleList.classList.add("is-hidden");

  const icon = title.parentNode.querySelector("i.mdi");
  icon.classList.remove("mdi-chevron-down");
  icon.classList.add("mdi-chevron-right");

  title.dataset.folded = "true";
}

function evExpandTOCTitle(title) {
  const subtitleList = title.nextElementSibling;
  subtitleList.classList.remove("is-hidden");

  const icon = title.parentNode.querySelector("i.mdi");
  icon.classList.remove("mdi-chevron-right");
  icon.classList.add("mdi-chevron-down");

  title.dataset.folded = "false";
}
