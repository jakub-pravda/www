// get the modal to use
var modal = document.getElementById("galleryModal");
// get all elements with class name "image fit"
var images = document.getElementsByClassName("image fit");

// set onclick event for each image from images, that will open the modal
for (var i = 0; i < images.length; i++) {
  images[i].onclick = function() {
    modal.style.display = "block";
    var imgElement = this.children[0];
    document.getElementById("img01").src = imgElement.src;
    document.getElementById("caption").innerHTML = imgElement.alt;
  }
}

// get the <span> element that closes the modal
var span = document.getElementsByClassName("modal")[0];

// when the user clicks on <span>, close the modal
span.onclick = function() {
  modal.style.display = "none";
} 