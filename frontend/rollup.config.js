import svelte from 'rollup-plugin-svelte';
import { nodeResolve } from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import typescript from '@rollup/plugin-typescript';
import livereload from 'rollup-plugin-livereload';
import { terser } from 'rollup-plugin-terser';
import css from 'rollup-plugin-css-only';
import fs from "fs";
import posthtml from "posthtml";
import { hash } from "posthtml-hash";
import copy from "rollup-plugin-copy";


const production = !process.env.ROLLUP_WATCH;
const BUILD_DIR = '../public';

// Inspired by https://github.com/metonym/svelte-rollup-template/blob/master/rollup.config.js
function hashStatic() {
	return {
		name: "hash-static",
		writeBundle() {
			const hashHandler = posthtml().use(
				// hashes `bundle.[custom-hash].css` and `bundle.[custom-hash].js`
				hash({
					path: BUILD_DIR,
					pattern: new RegExp(/\[custom-hash\]/),
					transformPath: (filepath) => filepath.replace('/public/', ""),
				}),
			)
			hashHandler.process(fs.readFileSync(`${BUILD_DIR}/index.html`))
				.then((result) =>
					fs.writeFileSync(`${BUILD_DIR}/index.html`, result.html)
				);
			hashHandler.process(fs.readFileSync(`${BUILD_DIR}/modify_frontend/modify.html`))
				.then((result) =>
					fs.writeFileSync(`${BUILD_DIR}/modify_frontend/modify.html`, result.html)
				);
		},
	};
}

export default {
	input: [
		'src/js/main.js',
	],
	output: {
		sourcemap: 'inline',
		format: 'iife',
		name: 'app',
		dir: BUILD_DIR,
		entryFileNames: '[name]-[custom-hash].js',
	},
	plugins: [
		copy({
			targets: [
				{ src: "src/index.html", dest: BUILD_DIR },
				{ src: "src/modify_frontend/*", dest: `${BUILD_DIR}/modify_frontend` },
				{ src: "src/assets", dest: `${BUILD_DIR}` },
				{ src: "node_modules/leaflet/dist/images", dest: `${BUILD_DIR}` },
			]
		}),
		svelte({
			compilerOptions: {
				// enable run-time checks when not in production
				dev: !production,
			}
		}),

		// we'll extract any component CSS out into
		// a separate file - better for performance
		css({
			output: 'bundle-[custom-hash].css',
		}),

		// If you have external dependencies installed from
		// npm, you'll most likely need these plugins. In
		// some cases you'll need additional configuration -
		// consult the documentation for details:
		// https://github.com/rollup/plugins/tree/master/packages/commonjs
		nodeResolve({
			browser: true,
			dedupe: ['svelte'],
			preferBuiltins: false
		}),
		commonjs(),
		typescript(),

		// In dev mode, call `npm run start` once
		// the bundle has been generated
		!production && serve(),

		// Watch the `public` directory and refresh the
		// browser on changes when not in production
		!production && livereload({watch: BUILD_DIR, delay: 100}),

		// If we're building for production (npm run build
		// instead of npm run dev), minify
		production && terser(),

		// Replace [custom-hash] in files with an actual hash
		production && hashStatic(),
	],
	watch: {
		clearScreen: false
	}
};

function serve() {
	let started = false;

	return {
		writeBundle() {
			if (!started) {
				started = true;

				require('child_process').spawn('npm', ['run', 'start', '--', '--dev'], {
					stdio: ['ignore', 'inherit', 'inherit'],
					shell: true
				});
			}
		}
	};
}
