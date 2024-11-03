// Gallery image hover
$( ".img-wrapper" ).hover(
  function() {
    $(this).find(".img-overlay").animate({opacity: 1}, 600);
  }, function() {
    $(this).find(".img-overlay").animate({opacity: 0}, 600);
  }
);

// Lightbox
var $overlay = $('<div id="overlay"></div>');
var $figure = $("<figure></figure>");
var $image = $("<img>");
var $caption = $('<figcaption></figcaption>');
var $prevButton = $('<div id="prevButton"><i class="fa fa-chevron-left"></i></div>');
var $nextButton = $('<div id="nextButton"><i class="fa fa-chevron-right"></i></div>');
var $exitButton = $('<div id="exitButton"><i class="fa fa-times"></i></div>');

var $imageWithCaption = $figure.append($image).append($caption);

// Add overlay
$overlay.append($imageWithCaption).prepend($prevButton).append($nextButton).append($exitButton);
$("#gallery").append($overlay);

// Hide overlay on default
$overlay.hide();

// Function to update the image and caption
function updateImageAndCaption(imageElement) {
  var imageLocation = imageElement.attr("src");
  var imageAltText = imageElement.attr("alt");
  $image.attr("src", imageLocation);
  $caption.text(imageAltText);
}

// When an image is clicked
$(".img-overlay").click(function(event) {
  // Prevents default behavior
  event.preventDefault();
  // Get the clicked image
  var $clickedImage = $(this).prev().find("img");
  // Update the image and caption
  updateImageAndCaption($clickedImage);
  // Fade in the overlay
  $overlay.fadeIn("slow");
});

// When the overlay is clicked
$overlay.click(function() {
  // Fade out the overlay
  $(this).fadeOut("slow");
});

// When next button is clicked
$nextButton.click(function(event) {
  // Hide the current image
  $("#overlay img").hide();
  // Overlay image location
  var $currentImgSrc = $("#overlay img").attr("src");
  // Image with matching location of the overlay image
  var $currentImg = $('#image-gallery img[src="' + $currentImgSrc + '"]');
  // Finds the next image
  var $nextImg = $($currentImg.closest(".image").next().find("img"));
  // All of the images in the gallery
  var $images = $("#image-gallery img");
  // If there is a next image
  if ($nextImg.length > 0) { 
    // Update the image and caption
    updateImageAndCaption($nextImg);
    // Fade in the next image
    $("#overlay img").fadeIn(800);
  } else {
    // Otherwise update the image and caption with the first image
    updateImageAndCaption($($images[0]));
    // Fade in the first image
    $("#overlay img").fadeIn(800);
  }
  // Prevents overlay from being hidden
  event.stopPropagation();
});

// When previous button is clicked
$prevButton.click(function(event) {
  // Hide the current image
  $("#overlay img").hide();
  // Overlay image location
  var $currentImgSrc = $("#overlay img").attr("src");
  // Image with matching location of the overlay image
  var $currentImg = $('#image-gallery img[src="' + $currentImgSrc + '"]');
  // Finds the previous image
  var $prevImg = $($currentImg.closest(".image").prev().find("img"));
  // All of the images in the gallery
  var $images = $("#image-gallery img");
  // If there is a previous image
  if ($prevImg.length > 0) { 
    // Update the image and caption
    updateImageAndCaption($prevImg);
    // Fade in the previous image
    $("#overlay img").fadeIn(800);
  } else {
    // Otherwise update the image and caption with the last image
    updateImageAndCaption($($images[$images.length - 1]));
    // Fade in the last image
    $("#overlay img").fadeIn(800);
  }
  // Prevents overlay from being hidden
  event.stopPropagation();
});

// When the exit button is clicked
$exitButton.click(function() {
  // Fade out the overlay
  $("#overlay").fadeOut("slow");
});