// Homepage-only: loads the hero background video (poster image shows
// immediately, video fades in once it can play) and a scroll-linked
// parallax on the terminal mockup (GSAP + ScrollTrigger). Both are
// progressive enhancements -- the hero reads perfectly without them.
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

function initHeroVideo() {
  const video = document.getElementById('hero-bg-video');
  if (!video) return;
  if (reduceMotion) return; // poster image alone is enough
  const saveData = navigator.connection && navigator.connection.saveData;
  if (saveData) return;

  video.addEventListener('playing', () => video.classList.add('is-visible'), { once: true });
  video.src = 'media/hero-bg.mp4';
  video.load();
  video.play().catch(() => {
    // Autoplay can be blocked; the poster image remains visible.
  });
}

async function initScrollParallax() {
  if (reduceMotion) return;
  const term = document.getElementById('hero-term');
  if (!term) return;
  try {
    const gsapModule = await import('https://esm.sh/gsap@3.12.5');
    const { default: ScrollTrigger } = await import('https://esm.sh/gsap@3.12.5/ScrollTrigger');
    const gsap = gsapModule.gsap || gsapModule.default;
    gsap.registerPlugin(ScrollTrigger);
    gsap.to(term, {
      yPercent: -8,
      ease: 'none',
      scrollTrigger: { trigger: term, start: 'top bottom', end: 'bottom top', scrub: 0.6 },
    });
  } catch (e) {
    // Parallax is a progressive enhancement.
  }
}

initHeroVideo();
initScrollParallax();
