import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/puppeteer',
  packages: [
    {
      importPath: 'github.com/malcolmston/puppeteer',
      name: 'puppeteer',
      synopsis: 'Package puppeteer is a standard-library-only page-automation toolkit for Go.',
      doc: 'Package puppeteer is a standard-library-only page-automation toolkit for Go.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Browser',
          signature: 'type Browser struct{}',
          doc: 'Browser owns an http.Client, a cookie jar and shared headers.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'Launch', signature: 'func Launch(opts *LaunchOptions) (*Browser, error)', doc: 'Launch creates a new Browser.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package puppeteer/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-Launch'), 'func Launch symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Browser'), 'type Browser symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
