import * as React from 'react';
import { cn } from '@/lib/utils';

export interface CurrencyInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  value: number;
  onValueChange: (amount: number) => void;
}

const CurrencyInput = React.forwardRef<HTMLInputElement, CurrencyInputProps>(
  ({ className, value, onValueChange, ...props }, ref) => {
    const formattedValue = value > 0 ? new Intl.NumberFormat('id-ID').format(value) : '';

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const digitsOnly = e.target.value.replace(/\D/g, '');
      if (!digitsOnly) {
        onValueChange(0);
        return;
      }
      const parsed = Number.parseInt(digitsOnly, 10);
      if (!Number.isNaN(parsed)) {
        onValueChange(parsed);
      }
    };

    return (
      <div className="relative flex items-center">
        <span className="pointer-events-none absolute left-3 text-sm font-mono font-semibold text-[#00E676]">
          Rp
        </span>
        <input
          ref={ref}
          type="text"
          inputMode="numeric"
          value={formattedValue}
          onChange={handleChange}
          placeholder="0"
          className={cn(
            'flex h-9 w-full rounded-md border border-input bg-transparent pl-10 pr-3 py-1 text-sm font-mono tabular-nums font-semibold shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
            className
          )}
          {...props}
        />
      </div>
    );
  }
);
CurrencyInput.displayName = 'CurrencyInput';

export { CurrencyInput };
