/* ============================================================
   Groove Addict Music — app logic
   ============================================================ */

/* ============================================================
   1. PRODUCT CATALOG — replace with real inventory
   ============================================================ */
const PRODUCTS = [
  { id:1,  name:"Mahogany Acoustic",          cat:"guitars",     price:299, badge:"Staff Pick",
    img:"https://images.unsplash.com/photo-1510915361894-db8b60106cb1?w=600&auto=format&fit=crop" },
  { id:2,  name:"Vintage-Style Strat",        cat:"guitars",     price:649, badge:null,
    img:"https://images.unsplash.com/photo-1564186763535-ebb21ef5277f?w=600&auto=format&fit=crop" },
  { id:3,  name:"4-String Jazz Bass",         cat:"guitars",     price:529, badge:null,
    img:"https://images.unsplash.com/photo-1612225330812-01a9c6b355ec?w=600&auto=format&fit=crop" },
  { id:4,  name:"88-Key Stage Piano",         cat:"keys",        price:899, badge:"Best Seller",
    img:"https://images.unsplash.com/photo-1520523839897-bd0b52f945a0?w=600&auto=format&fit=crop" },
  { id:5,  name:"61-Key Synth",               cat:"keys",        price:179, badge:null,
    img:"https://images.unsplash.com/photo-1571974599782-87624638275e?w=600&auto=format&fit=crop" },
  { id:6,  name:"5-Piece Drum Kit",           cat:"drums",       price:799, badge:null,
    img:"https://images.unsplash.com/photo-1519892300165-cb5542fb47c7?w=600&auto=format&fit=crop" },
  { id:7,  name:"Practice Snare + Stand",     cat:"drums",       price:189, badge:null,
    img:"https://images.unsplash.com/photo-1543443258-92b04ad5ec6b?w=600&auto=format&fit=crop" },
  { id:8,  name:"15W Tube Combo",             cat:"amps",        price:449, badge:"New",
    img:"https://images.unsplash.com/photo-1558098329-a11cff621064?w=600&auto=format&fit=crop" },
  { id:9,  name:"Multi-FX Pedal",             cat:"amps",        price:229, badge:null,
    img:"https://images.unsplash.com/photo-1601312378427-822b2b41da35?w=600&auto=format&fit=crop" },
  { id:10, name:"Premium Strings (3pk)",      cat:"accessories", price:24,  badge:null,
    img:"https://images.unsplash.com/photo-1558098329-a11cff621064?w=600&auto=format&fit=crop" },
  { id:11, name:"Adjustable Stand",           cat:"accessories", price:32,  badge:null,
    img:"https://images.unsplash.com/photo-1522075469751-3a6694fb2f61?w=600&auto=format&fit=crop" },
  { id:12, name:"Dynamic Vocal Mic",          cat:"accessories", price:99,  badge:null,
    img:"https://images.unsplash.com/photo-1590602847861-f357a9332bbc?w=600&auto=format&fit=crop" },
];

/* ============================================================
   2. STATE (in-memory only — no localStorage)
   ============================================================ */
let cart = [];
let activeFilter = "all";

const cap = s => s.charAt(0).toUpperCase() + s.slice(1);

/* ============================================================
   3. PRODUCT GRID + FILTERING
   ============================================================ */
const productGrid = document.getElementById("productGrid");
const filterBar   = document.getElementById("filterBar");

function renderProducts() {
  const list = activeFilter === "all" ? PRODUCTS : PRODUCTS.filter(p => p.cat === activeFilter);
  productGrid.innerHTML = list.map(p => `
    <div class="product">
      <div class="product-img" style="background-image:url('${p.img}')">
        ${p.badge ? `<span class="product-badge">${p.badge}</span>` : ""}
      </div>
      <div class="product-info">
        <div class="cat">${cap(p.cat)}</div>
        <h3>${p.name}</h3>
        <div class="price">$${p.price.toFixed(2)}</div>
        <button data-add-id="${p.id}">Add to cart</button>
      </div>
    </div>
  `).join("");
}

filterBar.addEventListener("click", e => {
  if (e.target.tagName !== "BUTTON") return;
  filterBar.querySelectorAll("button").forEach(b => b.classList.remove("active"));
  e.target.classList.add("active");
  activeFilter = e.target.dataset.filter;
  renderProducts();
});

productGrid.addEventListener("click", e => {
  const btn = e.target.closest("[data-add-id]");
  if (btn) addToCart(Number(btn.dataset.addId));
});

/* ============================================================
   4. CART
   ============================================================ */
const cartCountEl   = document.getElementById("cartCount");
const cartItemsEl   = document.getElementById("cartItems");
const cartTotalRow  = document.getElementById("cartTotalRow");
const cartTotalEl   = document.getElementById("cartTotal");
const checkoutBtn   = document.getElementById("checkoutBtn");
const cartModal     = document.getElementById("cartModal");
const cartOpenBtn   = document.getElementById("cartOpenBtn");
const cartCloseBtn  = document.getElementById("cartCloseBtn");

function addToCart(id) {
  const product = PRODUCTS.find(p => p.id === id);
  const existing = cart.find(i => i.id === id);
  if (existing) existing.qty++;
  else cart.push({ ...product, qty: 1 });
  updateCart();
  showToast(`${product.name} added`);
}

function removeFromCart(id) {
  cart = cart.filter(i => i.id !== id);
  updateCart();
}

function changeQty(id, delta) {
  const item = cart.find(i => i.id === id);
  if (!item) return;
  item.qty += delta;
  if (item.qty <= 0) removeFromCart(id);
  else updateCart();
}

function updateCart() {
  const count = cart.reduce((s, i) => s + i.qty, 0);
  cartCountEl.textContent = count;

  if (cart.length === 0) {
    cartItemsEl.innerHTML = '<div class="empty-cart">// Cart empty //</div>';
    cartTotalRow.style.display = "none";
    checkoutBtn.style.display = "none";
    return;
  }

  cartItemsEl.innerHTML = cart.map(i => `
    <div class="cart-item">
      <div class="cart-item-img" style="background-image:url('${i.img}')"></div>
      <div class="cart-item-info">
        <h4>${i.name}</h4>
        <div class="meta">$${i.price.toFixed(2)}</div>
      </div>
      <div class="qty-controls">
        <button data-qty-id="${i.id}" data-qty-delta="-1">&minus;</button>
        <span>${i.qty}</span>
        <button data-qty-id="${i.id}" data-qty-delta="1">+</button>
      </div>
    </div>
  `).join("");

  const total = cart.reduce((s, i) => s + i.price * i.qty, 0);
  cartTotalEl.textContent = "$" + total.toFixed(2);
  cartTotalRow.style.display = "flex";
  checkoutBtn.style.display = "block";
}

cartItemsEl.addEventListener("click", e => {
  const btn = e.target.closest("[data-qty-id]");
  if (btn) changeQty(Number(btn.dataset.qtyId), Number(btn.dataset.qtyDelta));
});

function openCart()  { cartModal.classList.add("active"); }
function closeCart() { cartModal.classList.remove("active"); }

cartOpenBtn.addEventListener("click", openCart);
cartCloseBtn.addEventListener("click", closeCart);
cartModal.addEventListener("click", e => { if (e.target === cartModal) closeCart(); });

checkoutBtn.addEventListener("click", () => {
  if (cart.length === 0) return;
  alert("Demo checkout!\n\nIn production this hooks into Stripe / Square / Shopify. Placeholder for now.");
  cart = [];
  updateCart();
  closeCart();
});

/* ============================================================
   5. FORMS
   ============================================================ */
const bookingForm = document.getElementById("bookingForm");
const contactForm = document.getElementById("contactForm");

if (bookingForm) {
  bookingForm.addEventListener("submit", e => {
    e.preventDefault();
    showToast("Lesson request sent — we'll be in touch");
    bookingForm.reset();
  });
}

if (contactForm) {
  contactForm.addEventListener("submit", e => {
    e.preventDefault();
    showToast("Message sent — thanks");
    contactForm.reset();
  });
}

/* ============================================================
   6. TOAST
   ============================================================ */
const toastEl = document.getElementById("toast");
let toastTimer;

function showToast(msg) {
  toastEl.textContent = msg;
  toastEl.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toastEl.classList.remove("show"), 2400);
}

/* ============================================================
   7. NAV / SMOOTH SCROLL / MOBILE MENU
   ============================================================ */
const navLinksEl = document.getElementById("navLinks");
const menuToggle = document.getElementById("menuToggle");

function closeMenu() { navLinksEl.classList.remove("open"); }
menuToggle.addEventListener("click", () => navLinksEl.classList.toggle("open"));

navLinksEl.querySelectorAll("a").forEach(a => a.addEventListener("click", closeMenu));

document.querySelectorAll("[data-scroll-to]").forEach(btn => {
  btn.addEventListener("click", () => {
    const id = btn.dataset.scrollTo;
    const el = document.getElementById(id);
    if (el) el.scrollIntoView({ behavior: "smooth" });
    closeMenu();
  });
});

/* ============================================================
   8. INIT
   ============================================================ */
renderProducts();
updateCart();
document.getElementById("year").textContent = new Date().getFullYear();
