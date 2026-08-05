/**
 * Join class names, dropping falsy entries.
 *
 * Lives in its own module rather than alongside the components so that files
 * exporting React components export only components — which is what keeps Fast
 * Refresh working during development.
 */
export function cx(...classes) {
  return classes.filter(Boolean).join(' ')
}
