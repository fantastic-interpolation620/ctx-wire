import {
  animate,
  motion,
  useMotionValue,
  useMotionValueEvent,
  useReducedMotion,
} from "motion/react";
import { useEffect, useState } from "react";

// AnimatedNumber counts up to `value` the first time it scrolls into view,
// formatting every frame so fine digits (cents, thousandths) visibly move. It
// re-animates when `value` changes later (e.g. refreshed stats), and respects
// reduced-motion by snapping straight to the final value with no animation.
export function AnimatedNumber({
  className,
  format,
  value,
}: {
  className?: string;
  format: (n: number) => string;
  value: number;
}) {
  const reduce = useReducedMotion();
  const mv = useMotionValue(reduce ? value : 0);
  const [display, setDisplay] = useState(() => format(reduce ? value : 0));
  const [inView, setInView] = useState(false);

  useMotionValueEvent(mv, "change", (v) => setDisplay(format(v)));

  useEffect(() => {
    if (!inView) return;
    if (reduce) {
      mv.set(value);
      return;
    }
    const controls = animate(mv, value, {
      duration: 1.4,
      ease: [0.16, 1, 0.3, 1],
    });
    return () => controls.stop();
  }, [inView, value, reduce, mv]);

  return (
    <motion.span
      className={className}
      onViewportEnter={() => setInView(true)}
      viewport={{ once: true, amount: 0.6 }}
    >
      {display}
    </motion.span>
  );
}
