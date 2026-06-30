

function initCardAnimations() {
  // ── Scroll-reveal with staggered delay ──────────────────────────────
  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) return;
      const card = entry.target as HTMLElement;
      const siblings = Array.from(card.parentElement?.children ?? []);
      const idx = siblings.indexOf(card);
      card.style.transitionDelay = `${(idx % 3) * 80}ms`;
      card.classList.add('revealed');
      observer.unobserve(card);
    });
  }, { threshold: 0.12 });

  function observeCards() {
    document.querySelectorAll<HTMLElement>('[data-reveal]').forEach(card => {
      if (!card.classList.contains('revealed')) observer.observe(card);
    });
  }
  observeCards();
  document.addEventListener('auctioncardsadded', observeCards);

  // ── 3D tilt + spotlight glow (only register once globally) ─────────
  if (!(window as any).__cardTiltInit) {
    (window as any).__cardTiltInit = true;

    const MAX_TILT = 6;
    let activeCard: HTMLElement | null = null;

    document.addEventListener('mousemove', (e) => {
      const card = (e.target as Element).closest<HTMLElement>('.auction-card');
      if (activeCard && activeCard !== card) {
        activeCard.style.transform = '';
        activeCard = null;
      }
      if (!card) return;
      activeCard = card;
      const r = card.getBoundingClientRect();
      const x = e.clientX - r.left;
      const y = e.clientY - r.top;
      const rotX = ((y - r.height / 2) / (r.height / 2)) * -MAX_TILT;
      const rotY = ((x - r.width / 2) / (r.width / 2)) * MAX_TILT;
      card.style.setProperty('--mouse-x', `${x}px`);
      card.style.setProperty('--mouse-y', `${y}px`);
      card.style.transition = 'border-color 0.2s ease, box-shadow 0.35s ease';
      card.style.transform = `perspective(800px) rotateX(${rotX}deg) rotateY(${rotY}deg) translateZ(4px)`;
    });

    document.addEventListener('mouseout', (e) => {
      const card = (e.target as Element).closest<HTMLElement>('.auction-card');
      if (!card) return;
      if (card.contains(e.relatedTarget as Node)) return;
      card.style.transition = 'border-color 0.2s ease, transform 0.5s cubic-bezier(0.16,1,0.3,1), box-shadow 0.35s ease';
      card.style.transform = '';
      activeCard = null;
    });
  }
}

// Run on initial load
initCardAnimations();
// Re-run after View Transition navigation
document.addEventListener('astro:page-load', initCardAnimations);
