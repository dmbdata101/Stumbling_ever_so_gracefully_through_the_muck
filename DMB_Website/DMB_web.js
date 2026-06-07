/* ============================================================
   D. Michael Brown — Journey to PhD
   App logic
   ============================================================ */

/* ============================================================
   1. CURATED CONTENT — edit these arrays to add photos/videos
   ============================================================ */

// To add a photo: push an object with `src` (relative path or URL) and `caption`.
// Drop image files in this folder (or a `/photos/` subfolder) and reference them.
// Example:
// { src: "photos/grad-day.jpg", caption: "Sophia Learning completion — March 2026" }
const PHOTOS = [
  // Add photos here as the journey unfolds.
];

// To add a video: push an object with `id` (the YouTube video ID — the part
// after `v=` or after `youtu.be/`) and an optional `caption`.
// Example:
// { id: "dQw4w9WgXcQ", caption: "Talk on psycho-oncology" }
const VIDEOS = [
  // Add videos here as the journey unfolds.
];

/* ============================================================
   2. ACTIVE NAV — highlight current section while scrolling
   ============================================================ */
const navLinks = document.querySelectorAll('.nav-links a');
const sectionEls = ['home', 'about', 'photos', 'videos', 'contact']
  .map(id => document.getElementById(id))
  .filter(Boolean);

function updateActiveNav() {
  const scrollPos = window.scrollY + 80;
  let current = 'home';
  sectionEls.forEach(sec => {
    if (sec.offsetTop <= scrollPos) current = sec.id;
  });
  navLinks.forEach(link => {
    link.classList.toggle('active', link.getAttribute('href') === '#' + current);
  });
}
window.addEventListener('scroll', updateActiveNav, { passive: true });

/* ============================================================
   3. PHOTOS — render the curated gallery
   ============================================================ */
const photosGrid = document.getElementById('photosGrid');

function renderPhotos() {
  if (!photosGrid) return;
  photosGrid.innerHTML = '';
  if (PHOTOS.length === 0) {
    photosGrid.innerHTML = '<p class="photos-empty">Photos coming soon as the journey unfolds.</p>';
    return;
  }
  PHOTOS.forEach(p => {
    const safeSrc = p.src.replace(/'/g, "\\'");
    const card = document.createElement('div');
    card.className = 'photo-card';
    card.innerHTML = `
      <img src="${p.src}" alt="${p.caption || 'Photo'}" onclick="openPhotoModal('${safeSrc}')" />
      <div class="caption">${p.caption || ''}</div>
    `;
    photosGrid.appendChild(card);
  });
}

function openPhotoModal(src) {
  const modal = document.getElementById('photoModal');
  const img = document.getElementById('modalImg');
  if (!modal || !img) return;
  img.src = src;
  modal.classList.add('active');
}

function closePhotoModal() {
  const modal = document.getElementById('photoModal');
  if (modal) modal.classList.remove('active');
}

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closePhotoModal();
});

/* ============================================================
   4. VIDEOS — render the curated gallery
   ============================================================ */
const videosGrid = document.getElementById('videosGrid');

function renderVideos() {
  if (!videosGrid) return;
  videosGrid.innerHTML = '';
  if (VIDEOS.length === 0) {
    videosGrid.innerHTML = '<p class="videos-empty">Videos coming soon as the journey unfolds.</p>';
    return;
  }
  VIDEOS.forEach(v => {
    const card = document.createElement('div');
    card.className = 'video-card';
    card.innerHTML = `
      <iframe src="https://www.youtube.com/embed/${v.id}" allowfullscreen
              title="${v.caption || 'Video'}"></iframe>
      ${v.caption ? `<div class="video-meta">${v.caption}</div>` : ''}
    `;
    videosGrid.appendChild(card);
  });
}

/* ============================================================
   5. EMAIL SIGNUP — Netlify forms via fetch
   ============================================================ */
const signupForm = document.getElementById('signupForm');
const signupSuccess = document.getElementById('signupSuccess');

if (signupForm) {
  signupForm.addEventListener('submit', async function(e) {
    e.preventDefault();
    const formData = new FormData(signupForm);
    try {
      const response = await fetch('/', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams(formData).toString()
      });
      if (response.ok) {
        signupForm.style.display = 'none';
        signupSuccess.classList.add('active');
      } else {
        alert('Something went wrong. Please try again or email psycd007@gmail.com directly.');
      }
    } catch (err) {
      alert('Network issue. Please try again or email psycd007@gmail.com directly.');
    }
  });
}

/* ============================================================
   6. INIT
   ============================================================ */
renderPhotos();
renderVideos();
updateActiveNav();
