/**
 * FBT (Frequently Bought Together) quick-add handler.
 *
 * On clicking + Add on an FBT chip:
 *   1. POST /cart with Accept: application/json (no redirect)
 *   2. Update cart badge and Cart(N) heading
 *   3. Update or insert the cart item row inline (no page reload)
 *   4. Update cart total locally using Money arithmetic on data attributes
 *   5. Fade-remove the clicked FBT chip
 *   6. Fade-remove the same product from the "You May Also Like" strip
 */

function addMoneyUnitsNanos(aUnits, aNanos, bUnits, bNanos) {
  let nanos = aNanos + bNanos;
  let carry = Math.trunc(nanos / 1_000_000_000);
  nanos = nanos % 1_000_000_000;
  if (nanos < 0) {
    nanos += 1_000_000_000;
    carry -= 1;
  }
  return { units: aUnits + bUnits + carry, nanos };
}

function formatMoney(units, nanos, currency) {
  const amount = units + nanos / 1_000_000_000;
  try {
    return new Intl.NumberFormat(navigator.language, {
      style: 'currency',
      currency: currency,
    }).format(amount);
  } catch (_) {
    return `${currency} ${amount.toFixed(2)}`;
  }
}

/** Multiply a unit price by a whole quantity (nanos stay in [0, 1e9)). */
function multiplyMoney(priceUnits, priceNanos, qty) {
  const totalNanos = priceNanos * qty;
  const carry = Math.trunc(totalNanos / 1_000_000_000);
  const nanos = ((totalNanos % 1_000_000_000) + 1_000_000_000) % 1_000_000_000;
  return { units: priceUnits * qty + carry, nanos };
}

function fadeRemove(el) {
  if (!el) return;
  el.style.transition = 'opacity 0.3s';
  el.style.opacity = '0';
  setTimeout(() => el.remove(), 300);
}

document.addEventListener('click', async (e) => {
  // --- Remove from cart ---
  const removeBtn = e.target.closest('.cart-remove-btn');
  if (removeBtn) {
    e.preventDefault();
    removeBtn.disabled = true;

    const productId  = removeBtn.dataset.productId;
    const priceUnits = parseInt(removeBtn.dataset.priceUnits, 10);  // line total
    const priceNanos = parseInt(removeBtn.dataset.priceNanos, 10);  // line total
    const currency   = removeBtn.dataset.currency;

    try {
      const res = await fetch('/cart/remove', {
        method: 'POST',
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({ product_id: productId }),
      });
      if (!res.ok) throw new Error(res.status);
      const { cart_size } = await res.json();

      // Update badge and heading
      document.querySelectorAll('.cart-size-circle').forEach((el) => {
        el.textContent = cart_size;
      });
      const cartHeading = document.getElementById('cart-heading');
      if (cartHeading) cartHeading.textContent = `Cart (${cart_size})`;

      // Subtract line total from cart total display.
      // priceUnits/priceNanos are already the line total (unit price × quantity)
      // from the server-rendered data-price-* attributes — do NOT multiply again.
      const totalEl = document.getElementById('cart-total-display');
      if (totalEl) {
        const cur = parseInt(totalEl.dataset.units, 10);
        const curN = parseInt(totalEl.dataset.nanos, 10);
        const { units, nanos } = addMoneyUnitsNanos(cur, curN, -priceUnits, -priceNanos);
        totalEl.dataset.units = units;
        totalEl.dataset.nanos = nanos;
        totalEl.textContent = formatMoney(units, nanos, currency);
      }

      // Remove the item row and its FBT strip
      const row = document.querySelector(`.cart-summary-item-row[data-product-id="${productId}"]`);
      const strip = row && row.nextElementSibling && row.nextElementSibling.classList.contains('fbt-inline-strip')
        ? row.nextElementSibling : null;
      fadeRemove(strip);
      fadeRemove(row);

      // If no items left, reload to show empty cart state
      if (cart_size === 0) setTimeout(() => location.reload(), 350);
    } catch (_) {
      removeBtn.disabled = false;
    }
    return;
  }
});

document.addEventListener('click', async (e) => {
  const btn = e.target.closest('.fbt-add-btn');
  if (!btn) return;
  e.preventDefault();

  const chip        = btn.closest('.fbt-chip');
  const productId   = btn.dataset.productId;
  const priceUnits  = parseInt(btn.dataset.priceUnits, 10);
  const priceNanos  = parseInt(btn.dataset.priceNanos, 10);
  const currency    = btn.dataset.currency;

  btn.disabled = true;

  const body = new URLSearchParams({ product_id: productId, quantity: '1' });

  try {
    const res = await fetch('/cart', {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body,
    });
    if (!res.ok) throw new Error(res.status);
    const { cart_size, product_id, quantity, name, picture } = await res.json();

    // 1. Update cart badge
    document.querySelectorAll('.cart-size-circle').forEach((el) => {
      el.textContent = cart_size;
    });

    // 2. Update Cart(N) heading
    const cartHeading = document.getElementById('cart-heading');
    if (cartHeading) cartHeading.textContent = `Cart (${cart_size})`;

    // 3. Update cart items section inline
    const existingRow = document.querySelector(
      `.cart-summary-item-row[data-product-id="${product_id}"]`
    );
    if (existingRow) {
      // Product was already in cart — update quantity and line price
      const qtyEl = existingRow.querySelector('.cart-item-quantity');
      if (qtyEl) qtyEl.textContent = `Quantity: ${quantity}`;
      const priceEl = existingRow.querySelector('.cart-item-price');
      if (priceEl) {
        const { units, nanos } = multiplyMoney(priceUnits, priceNanos, quantity);
        priceEl.textContent = formatMoney(units, nanos, currency);
      }
    } else {
      // New product — insert a new cart item row before the shipping row
      const shippingRow = document.querySelector('.cart-summary-shipping-row');
      if (shippingRow) {
        const lineMoney = formatMoney(priceUnits, priceNanos, currency);
        const newRow = document.createElement('div');
        newRow.className = 'row cart-summary-item-row';
        newRow.dataset.productId = product_id;
        newRow.innerHTML = `
          <div class="col-md-4 pl-md-0">
            <img class="img-fluid" alt="" src="${picture}"/>
          </div>
          <div class="col-md-8 pr-md-0">
            <div class="row"><div class="col"><h4>${name}</h4></div></div>
            <div class="row cart-summary-item-row-item-id-row">
              <div class="col">SKU #${product_id}</div>
            </div>
            <div class="row">
              <div class="col cart-item-quantity">Quantity: ${quantity}</div>
              <div class="col pr-md-0 text-right">
                <strong class="cart-item-price">${lineMoney}</strong>
              </div>
            </div>
            <div class="row mt-1">
              <div class="col">
                <button class="cart-remove-btn"
                        data-product-id="${product_id}"
                        data-price-units="${priceUnits}"
                        data-price-nanos="${priceNanos}"
                        data-currency="${currency}"
                        style="background:none;border:none;padding:0;color:#c0392b;font-size:0.8em;cursor:pointer;text-decoration:underline;">
                  Remove
                </button>
              </div>
            </div>
          </div>`;
        shippingRow.parentNode.insertBefore(newRow, shippingRow);
      }
      // Reload so server can render FBT strips for the newly added item
      setTimeout(() => location.reload(), 400);
    }

    // 4. Update cart total locally
    const totalEl = document.getElementById('cart-total-display');
    if (totalEl) {
      const currentUnits = parseInt(totalEl.dataset.units, 10);
      const currentNanos = parseInt(totalEl.dataset.nanos, 10);
      const { units, nanos } = addMoneyUnitsNanos(
        currentUnits, currentNanos, priceUnits, priceNanos
      );
      totalEl.dataset.units = units;
      totalEl.dataset.nanos = nanos;
      totalEl.textContent = formatMoney(units, nanos, currency);
    }

    // 5. Fade-remove the clicked FBT chip
    fadeRemove(chip);

    // 6. Fade-remove same product from "You May Also Like" strip
    const recCard = document.querySelector(
      `#recommendations-strip [data-product-id="${product_id}"]`
    );
    fadeRemove(recCard);

  } catch (_) {
    btn.disabled = false;
    if (chip) chip.classList.add('fbt-error');
  }
});
