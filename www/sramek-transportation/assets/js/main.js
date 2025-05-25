document.addEventListener('DOMContentLoaded', () => {
  "use strict";

  /**
   * Preloader
   */
  const preloader = document.querySelector('#preloader');
  if (preloader) {
    window.addEventListener('load', () => {
      preloader.remove();
    });
  }

  /**
   * Sticky header on scroll
   */
  const selectHeader = document.querySelector('#header');
  if (selectHeader) {
    document.addEventListener('scroll', () => {
      window.scrollY > 100 ? selectHeader.classList.add('sticked') : selectHeader.classList.remove('sticked');
    });
  }

  /**
   * Scroll top button
   */
  const scrollTop = document.querySelector('.scroll-top');
  if (scrollTop) {
    const togglescrollTop = function() {
      window.scrollY > 100 ? scrollTop.classList.add('active') : scrollTop.classList.remove('active');
    }
    window.addEventListener('load', togglescrollTop);
    document.addEventListener('scroll', togglescrollTop);
    scrollTop.addEventListener('click', window.scrollTo({
      top: 0,
      behavior: 'smooth'
    }));
  }

  /**
   * Mobile nav toggle
   */
  const mobileNavShow = document.querySelector('.mobile-nav-show');
  const mobileNavHide = document.querySelector('.mobile-nav-hide');

  document.querySelectorAll('.mobile-nav-toggle').forEach(el => {
    el.addEventListener('click', function(event) {
      event.preventDefault();
      mobileNavToogle();
    })
  });

  function mobileNavToogle() {
    document.querySelector('body').classList.toggle('mobile-nav-active');
    mobileNavShow.classList.toggle('d-none');
    mobileNavHide.classList.toggle('d-none');
  }

  /**
   * Hide mobile nav on same-page/hash links
   */
  document.querySelectorAll('#navbar a').forEach(navbarlink => {

    if (!navbarlink.hash) return;

    let section = document.querySelector(navbarlink.hash);
    if (!section) return;

    navbarlink.addEventListener('click', () => {
      if (document.querySelector('.mobile-nav-active')) {
        mobileNavToogle();
      }
    });

  });

  /**
   * Toggle mobile nav dropdowns
   */
  const navDropdowns = document.querySelectorAll('.navbar .dropdown > a');

  navDropdowns.forEach(el => {
    el.addEventListener('click', function(event) {
      if (document.querySelector('.mobile-nav-active')) {
        event.preventDefault();
        this.classList.toggle('active');
        this.nextElementSibling.classList.toggle('dropdown-active');

        let dropDownIndicator = this.querySelector('.dropdown-indicator');
        dropDownIndicator.classList.toggle('bi-chevron-up');
        dropDownIndicator.classList.toggle('bi-chevron-down');
      }
    })
  });

  /**
   * Initiate glightbox
   */
  const glightbox = GLightbox({
    selector: '.glightbox',
    preload: false
  });

  /**
   * Init swiper slider with 1 slide at once in desktop view
   */
  new Swiper('.slides-1', {
    speed: 600,
    loop: true,
    autoplay: {
      delay: 5000,
      disableOnInteraction: false
    },
    slidesPerView: 'auto',
    pagination: {
      el: '.swiper-pagination',
      type: 'bullets',
      clickable: true
    },
    navigation: {
      nextEl: '.swiper-button-next',
      prevEl: '.swiper-button-prev',
    }
  });
});

/**
 * Header and Footer custom elements
 
 */
class Header extends HTMLElement {
  connectedCallback() {
    this.innerHTML = `
      <header id="header" class="header d-flex align-items-center fixed-top">
        <div class="container-fluid container-xl d-flex align-items-center justify-content-between">

          <a href="index.html" class="logo d-flex align-items-center">
            <!-- Uncomment the line below if you also wish to use an image logo -->
            <!-- <img class="img-fluid img-thumbnail" src="images/logo.png" alt=""> -->
            <h1>Jiří Šrámek</h1>
          </a>

          <i class="mobile-nav-toggle mobile-nav-show bi bi-list"></i>
          <i class="mobile-nav-toggle mobile-nav-hide d-none bi bi-x"></i>
          <nav id="navbar" class="navbar">
            <ul>
              <li><a href="index.html" class="active">Domů</a></li>
              <li><a href="cenik.html">Ceník služeb</a></li>
              <li class="dropdown"><a href="#"><span>Služby</span> <i class="bi bi-chevron-down dropdown-indicator"></i></a>
                <ul>
                  <li><a href="pronajemkontejneru.html">Pronájem kontejnerů</a></li>
                  <li><a href="zemniprace.html">Zemní práce</a></li>
                  <li><a href="deponie.html">Doprava materiálu a deponie</a></li>
                  <li><a href="recyklace.html">Recyklace</a></li>
                  <li><a href="udrzbyzahrad.html">Údržby zahrad</a></li>
                  <li><a href="zahradnictvi.html">Zahradnictví</a></li>
                </ul>
              </li>
              <li><a href="flotila.html">Vozový park</a></li>
              <li><a class="get-a-quote" href="kontakt.html">Kontakt</a></li>
            </ul>
          </nav><!-- .navbar -->

        </div>
      </header>
    `;
  }
}

class Footer extends HTMLElement {
  connectedCallback() {
    this.innerHTML = `    
      <footer id="footer" class="footer">
        <div class="container">
          <div class="row gy-4">
            <div class="col-lg-5 col-md-12 footer-info">
              <a href="index.html" class="logo d-flex align-items-center">
                <span>Šrámek autodoprava</span>
              </a>
              <p>Váš spolehlivý partner v oblasti autodopravy a zemních prací</p>
              <div class="social-links d-flex mt-4">
                <a href="https://www.youtube.com/@jirisramek4838" class="youtube" target="_blank"><i class="bi bi-youtube"></i></a>
                <a href="https://www.facebook.com/p/Autodoprava-%C5%A0r%C3%A1mek-100084000835529" class="facebook" target="_blank"><i class="bi bi-facebook"></i></a>
                <a href="mailto:info@sramek-autodoprava.cz" class="email"><i class="bi bi-envelope"></i></a>
                <a href="tel:+420603484033" class="phone"><i class="bi bi-telephone-plus"></i></a>
              </div>
            </div>

            <div class="col-lg-3 col-md-12 footer-contact text-center text-md-start">
              <h4>Adresa</h4>
              <p>
                Mělnická 964/44<br>
                Brandýs nad Labem - Stará Boleslav<br>
                250 01 <br><br>
              </p>

            </div>

            <div class="col-lg-3 col-md-12 footer-contact text-center text-md-start">
              <h4>Kontaktujte nás</h4>
              <p>
                <strong>Telefon:</strong> <a href="tel:+420603484033">+420 603 484 033</a><br>
                <strong>Email:</strong> <a href="mailto:info@sramek-autodoprava.cz">info@sramek-autodoprava.cz</a><br>
              </p>

            </div>

          </div>
        </div>

        <div class="container mt-4">
          <div class="copyright">
            &copy; 2025 <strong><span>Šrámek autodoprava</span></strong>
          </div>
          <div class="credits">
            Designed by <a href="https://bootstrapmade.com/">BootstrapMade</a>
          </div>
        </div>

      </footer>  
    `;
  }
}

class SiteDescription extends HTMLElement {
  connectedCallback() {
    this.innerHTML = `
    <meta name="keywords" content="zemní práce, kontejnery, autodoprava, šrámek, stará boleslav, brandýs nad labem, bagrování, základy staveb, recyklace, deponie">
    <meta name="description" content="Profesionální zemní práce, pronájem odpadních kontejnerů, deponie a rozvoz materiálu, údržba zahrad v Brandýs nad Labem-Stará Boleslav. Rychlé a spolehlivé služby.">
    `;
  }
}

customElements.define('main-header', Header);
customElements.define('main-footer', Footer);
customElements.define('main-description', SiteDescription);
