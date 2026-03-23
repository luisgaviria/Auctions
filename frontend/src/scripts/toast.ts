export function showToast(message: string, type: 'info' | 'success' | 'error' = 'info'): void {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.setAttribute('role', 'alert');
  const icon = document.createElement('i');
  icon.className = `fas fa-${type === 'success' ? 'check-circle' : type === 'error' ? 'exclamation-circle' : 'info-circle'}`;
  toast.appendChild(icon);
  toast.appendChild(document.createTextNode(' ' + message));
  document.body.appendChild(toast);
  setTimeout(() => {
    toast.classList.add('show');
    setTimeout(() => {
      toast.classList.remove('show');
      setTimeout(() => document.body.removeChild(toast), 300);
    }, 3000);
  }, 100);
}
