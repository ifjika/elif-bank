/**
 * Formats an integer amount into Indonesian Rupiah currency string.
 * Example: 50000000 -> "Rp 50.000.000"
 */
export function formatRupiah(amount: number): string {
  const isNegative = amount < 0;
  const absAmount = Math.abs(amount);
  const formatted = new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(absAmount);

  return isNegative ? `-${formatted}` : formatted;
}
