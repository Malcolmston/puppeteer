import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NodeVsGo } from '../../../src/components/NodeVsGo';
import { PUPPETEER } from '../../../src/data';

describe('NodeVsGo', () => {
  it('renders the comparison heading and both Node.js and Go columns', () => {
    const { container } = render(<NodeVsGo lib={PUPPETEER} />);
    expect(container.querySelector(`#${PUPPETEER.id}-cmp`)).not.toBeNull();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(container.querySelectorAll('.compare .code').length).toBe(2);
  });
});
