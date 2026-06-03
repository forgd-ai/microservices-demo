/**
 * FBT (Frequently Bought Together) quick-add handler.
 *
 * On clicking + Add to Cart on an FBT chip:
 *   1. POST /cart with Accept: application/json (no redirect)
 *   2. Update cart badge count in header
 *   3. Update cart total display locally using Money arithmetic on data attributes
 *   4. Fade-remove the clicked FBT chip
 *   5. Fade-remove the same product from the "You May Also Like" strip
 */

/**
 * Adds two Money values represented as {units, nanos}.
 * Handles nanos overflow (carry into units) and negative nanos normalisation.
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

/**
 * Formats a Money value for display using the browser locale.
 * Matches the currency symbol/format the server renders via renderMoney.
 */
function formatMoney(units, nanos, currency) {
  const amount = units + nanos / 1_000_000_000;
  try {
    return new Intl.NumberFormat(navigator.language, {
      style: 'currency',
      currency: currency,
    }).format(amount);
  } catch (_) {
    // Fallback if currency code is unrecognised
    return `${currency} ${amount.toFixed(2)}`;
  }
}

function fadeRemove(el) {
  if (!el) return;
  el.style.transition = 'opacity 0.3s';
  el.style.opacity = '0';
  setTimeout(() => el.remove(), 300);
}

document.addEventListener('click', async (e) => {
  const btn = e.target.closest('.fbt-add-btn');
  if (!btn) return;
  e.preventDefault();

  const colWrap   = btn.closest('[class*="col"]');
  const chip      = btn.closest('.fbt-chip');
  const productId = btn.dataset.productId;
  const priceUnits = parseInt(btn.dataset.priceUnits, 10);
  const priceNanos = parseInt(btn.dataset.priceNanos, 10);
  const currency   = btn.dataset.currency;

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
    const { cart_size } = await res.json();

    // 1. Update cart badge
    document.querySelectorAll('.cart-size-circle').forEach((el) => {
      el.textContent = cart_size;
    });

    // 2. Update cart total locally
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

    // 3. Remove the clicked FBT chip column
    fadeRemove(colWrap || chip);

    // 4. Remove same product from "You May Also Like" strip
    const recCard = document.querySelector(
      `#recommendations-strip [data-product-id="${productId}"]`
    );
    fadeRemove(recCard);

  } catch (_) {
    btn.disabled = false;
    if (chip) chip.classList.add('fbt-error');
  }
});
