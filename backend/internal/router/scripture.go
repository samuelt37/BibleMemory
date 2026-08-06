package router

import (
    "github.com/go-chi/chi/v5"
    "github.com/samuelt37/BibleMemory/internal/handler"
)

func RegisterScriptureRoutes(
    r chi.Router,
    scriptureHandler *handler.ScriptureHandler,
) {
    r.Get(
        "/scripture/{translation}/{book}",
        scriptureHandler.GetBook,
    )

    //r.Get(
    //    "/scripture/{translation}/{book}/{chapter}",
    //    scriptureHandler.GetChapter,
    //)

    //r.Get(
    //    "/scripture/{translation}/{book}/{chapter}/{verse}",
    //    scriptureHandler.GetVerse,
    //)
}
