/*
	Spectral by HTML5 UP
	html5up.net | @ajlkn
	Free for personal and commercial use under the CCA 3.0 license (html5up.net/license)
*/

/**
 * Header elements
 */

class Header extends HTMLElement {
	/*remark: this header element must be defined before the main fuction! Main function works with the #menu element, which have to be properly inserted before!*/ 
	connectedCallback() {
    this.innerHTML = `
	    <!-- Header -->
		<header id="header" class="alt">
		<h1><a href="index.html">Zahradnictví Šrámek</a></h1>
		<nav id="nav">
			<ul>
			<li class="special">
				<a href="#menu" class=
				"menuToggle"><span>Menu</span></a>
				<div id="menu">
				<ul>
					<li>
					<a href="index.html">Domů</a>
					</li>
					<li>
					<a href="gallery.html">Galerie</a>
					</li>
					<li>
					<a href="goods.html">Sortiment</a>
					</li>
					<li>
					<a href="svatby.html">Svatební kytice</a>
					</li>
					<li>
					<a href="udrzby.html">Údržby zahrad</a>
					</li>
					<li>
					<a href="#three">Kontakt</a>
					</li>
				</ul>
				</div>
			</li>
			</ul>
		</nav>
		</header><!-- Banner -->
    `;
  }
}

customElements.define('main-header', Header);

(function($) {

	var	$window = $(window),
		$body = $('body'),
		$wrapper = $('#page-wrapper'),
		$banner = $('#banner'),
		$header = $('#header');

	// Breakpoints.
		breakpoints({
			xlarge:   [ '1281px',  '1680px' ],
			large:    [ '981px',   '1280px' ],
			medium:   [ '737px',   '980px'  ],
			small:    [ '481px',   '736px'  ],
			xsmall:   [ null,      '480px'  ]
		});

	// Play initial animations on page load.
		$window.on('load', function() {
			window.setTimeout(function() {
				$body.removeClass('is-preload');
			}, 100);
		});

	// Mobile?
		if (browser.mobile)
			$body.addClass('is-mobile');
		else {

			breakpoints.on('>medium', function() {
				$body.removeClass('is-mobile');
			});

			breakpoints.on('<=medium', function() {
				$body.addClass('is-mobile');
			});

		}

	// Scrolly.
		$('.scrolly')
			.scrolly({
				speed: 1500,
				offset: $header.outerHeight()
			});

	// Menu.
		$('#menu')
			.append('<a href="#menu" class="close"></a>')
			.appendTo($body)
			.panel({
				delay: 500,
				hideOnClick: true,
				hideOnSwipe: true,
				resetScroll: true,
				resetForms: true,
				side: 'right',
				target: $body,
				visibleClass: 'is-menu-visible'
			});

	// Header.
		if ($banner.length > 0
		&&	$header.hasClass('alt')) {

			$window.on('resize', function() { $window.trigger('scroll'); });

			$banner.scrollex({
				bottom:		$header.outerHeight() + 1,
				terminate:	function() { $header.removeClass('alt'); },
				enter:		function() { $header.addClass('alt'); },
				leave:		function() { $header.removeClass('alt'); }
			});

		}

	// Initiate glightbox
		const glightbox = GLightbox({
			selector: '.glightbox',
			preload: false
		});

})(jQuery);