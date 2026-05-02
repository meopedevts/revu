export default {
  multipass: true,
  plugins: [
    {
      name: "preset-default",
      params: {
        overrides: {
          cleanupIds: { minify: true },
        },
      },
    },
    "removeDimensions",
  ],
}
