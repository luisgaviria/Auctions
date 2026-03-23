export function initDrawer(): void {
  const drawer    = document.getElementById('mobile-drawer');
  const toggleBtn = document.getElementById('drawer-toggle');
  const label     = document.getElementById('drawer-label');
  if (!drawer || !toggleBtn || !label) return;

  toggleBtn.addEventListener('click', () => {
    const expanded = drawer.dataset.state === 'expanded';
    drawer.dataset.state = expanded ? 'peek' : 'expanded';
    label.textContent    = expanded ? 'Show listings' : 'Hide listings';
  });

  let touchStartY = 0;
  drawer.addEventListener('touchstart', (e) => { touchStartY = e.touches[0].clientY; }, { passive: true });
  drawer.addEventListener('touchend', (e) => {
    const delta = touchStartY - e.changedTouches[0].clientY;
    if (delta >  40) { drawer.dataset.state = 'expanded'; label.textContent = 'Hide listings'; }
    if (delta < -40) { drawer.dataset.state = 'peek';     label.textContent = 'Show listings'; }
  }, { passive: true });
}
