// package i18n processes ui text translations.
// howto:(see https://www.alexedwards.net/blog/i18n-managing-translations)
// It's important to emphasize that you don't edit this file in place. Instead,
// the workflow for adding a translation goes like this:
//
// 1. You generate the out.gotext.json files containing the messages which need
// to be translated (which we've just done).
// 2. You send these files to a translator, who edits the JSON to include the necessary translations.
// They then send the updated files back to you.
// 3. You then save these updated files with the name messages.gotext.json in the folder for the appropriate language.
package i18n
