// Site-wide motion layer: smooth scroll (Lenis) + scroll-reveal
// (IntersectionObserver) + light hover/press feedback (Motion One).
// Loaded on every page. Heavier, page-specific choreography (GSAP,
// Three.js) lives in the pages that actually need it (currently just
// the homepage hero) and is not part of this shared module.
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

function initReveal() {
  const targets = document.querySelectorAll('.reveal, .reveal-stagger');
  if (!targets.length) return;
  if (reduceMotion || !('IntersectionObserver' in window)) {
    targets.forEach((el) => el.classList.add('is-visible'));
    return;
  }
  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add('is-visible');
        observer.unobserve(entry.target);
      }
    });
  }, { threshold: 0.15, rootMargin: '0px 0px -8% 0px' });
  targets.forEach((el) => observer.observe(el));
}

async function initLenis() {
  if (reduceMotion) return;
  try {
    const { default: Lenis } = await import('https://esm.sh/lenis@1');
    const lenis = new Lenis({ duration: 1.05, smoothWheel: true });
    function raf(time) {
      lenis.raf(time);
      requestAnimationFrame(raf);
    }
    requestAnimationFrame(raf);
  } catch (e) {
    // Smooth scroll is a progressive enhancement; native scroll still works.
  }
}

async function initHoverFeedback() {
  if (reduceMotion) return;
  try {
    const { animate, hover, press } = await import('https://esm.sh/motion@11');
    const interactive = document.querySelectorAll('.btn, .card.hoverable, .badge-card');
    interactive.forEach((el) => {
      hover(el, () => {
        animate(el, { scale: 1.015 }, { duration: 0.18, easing: [0.16, 1, 0.3, 1] });
        return () => animate(el, { scale: 1 }, { duration: 0.18, easing: [0.16, 1, 0.3, 1] });
      });
      press(el, () => {
        animate(el, { scale: 0.98 }, { duration: 0.1 });
        return () => animate(el, { scale: 1 }, { duration: 0.15 });
      });
    });
  } catch (e) {
    // Micro-interactions are a progressive enhancement.
  }
}

let lucideRef = null;
export async function refreshIcons() {
  try {
    if (!lucideRef) lucideRef = await import('https://esm.sh/lucide@latest');
    lucideRef.createIcons({ icons: lucideRef.icons, attrs: { 'stroke-width': 1.75 } });
  } catch (e) {
    // Icons are progressive enhancement; markup still has readable fallbacks.
  }
}

initReveal();
initLenis();
initHoverFeedback();
refreshIcons();
