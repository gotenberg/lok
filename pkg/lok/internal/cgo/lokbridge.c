#include "lokbridge.h"

#include <stdbool.h>
#include <string.h>

// Values from LibreOfficeKitEnums.h, which cannot be included from C: it
// defines a C++-only helper (lokCallbackTypeToString) outside any __cplusplus
// guard. Both values are part of the LibreOfficeKit ABI and never change.
#define LOK_BRIDGE_CALLBACK_DOCUMENT_PASSWORD 20
#define LOK_BRIDGE_FEATURE_DOCUMENT_PASSWORD (1ULL << 0)

LoKit* lok_bridge_init_2(const char* installPath,
                         const char* userProfilePath) {
    return lok_init_2(installPath, userProfilePath);
}

void lok_bridge_destroy(LoKit* pOffice) {
    if (!pOffice) return;
    pOffice->pClass->destroy(pOffice);
}

// LibreOfficeKit asks for a document password through the registered callback
// and blocks until setDocumentPassword answers. The callback runs synchronously
// on the loading thread with the SolarMutex released, so answering from inside
// it is safe. A NULL password aborts the load. Only the first request of a load
// gets the password: a wrong password makes LibreOffice ask again, and
// answering NULL then stops the loop.
static void lok_bridge_callback(int nType, const char* pPayload, void* pData) {
    PasswordState* pState = (PasswordState*)pData;
    if (!pState || !pPayload || nType != LOK_BRIDGE_CALLBACK_DOCUMENT_PASSWORD) return;

    const char* pPassword = NULL;
    if (pState->nRequests == 0) pPassword = pState->pPassword;
    pState->nRequests++;

    pState->pOffice->pClass->setDocumentPassword(pState->pOffice, pPayload,
                                                 pPassword);
}

PasswordState* lok_bridge_enable_document_password(LoKit* pOffice) {
    if (!pOffice) return NULL;

    PasswordState* pState = calloc(1, sizeof(PasswordState));
    if (!pState) return NULL;
    pState->pOffice = pOffice;

    pOffice->pClass->registerCallback(pOffice, lok_bridge_callback, pState);
    pOffice->pClass->setOptionalFeatures(pOffice,
                                         LOK_BRIDGE_FEATURE_DOCUMENT_PASSWORD);

    return pState;
}

void lok_bridge_free_password_state(PasswordState* pState) {
    free(pState);
}

char* lok_bridge_get_error(LoKit* pOffice) {
    if (!pOffice) return NULL;
    return pOffice->pClass->getError(pOffice);
}

void lok_bridge_free_error(LoKit* pOffice, char* pErr) {
    if (!pOffice || !pErr) return;
    if (LIBREOFFICEKIT_HAS(pOffice, freeError)) {
        pOffice->pClass->freeError(pErr);
    } else {
        free(pErr);
    }
}

char* lok_bridge_get_version_info(LoKit* pOffice) {
    if (!pOffice) return NULL;
    return pOffice->pClass->getVersionInfo(pOffice);
}

// LOK runMacro returns 0 on failure, unlike typical C convention.
int lok_bridge_run_macro(LoKit* pOffice, const char* pURL) {
    if (!pOffice) return 0;
    return pOffice->pClass->runMacro(pOffice, pURL);
}

// trimMemory was added in LibreOffice 7.6. Older LibreOfficeKit headers do not
// declare the struct member, so referencing it (even through LIBREOFFICEKIT_HAS,
// which uses offsetof) fails to compile. Build with -DLOK_HAS_TRIM_MEMORY (via
// CGO_CFLAGS) when compiling against LibreOffice 7.6+ headers to enable the
// feature; otherwise these functions degrade to no-ops. When the member is
// present, LIBREOFFICEKIT_HAS still guards against an older runtime.
int lok_bridge_has_trim_memory(LoKit* pOffice) {
#ifdef LOK_HAS_TRIM_MEMORY
    if (!pOffice) return 0;
    return LIBREOFFICEKIT_HAS(pOffice, trimMemory);
#else
    (void)pOffice;
    return 0;
#endif
}

void lok_bridge_trim_memory(LoKit* pOffice, int nTarget) {
#ifdef LOK_HAS_TRIM_MEMORY
    if (pOffice && LIBREOFFICEKIT_HAS(pOffice, trimMemory)) {
        pOffice->pClass->trimMemory(pOffice, nTarget);
    }
#else
    (void)pOffice;
    (void)nTarget;
#endif
}

// Without a password, a password request is answered with NULL and the load
// fails instead of blocking.
LoKitDocument* lok_bridge_document_load_with_password(LoKit* pOffice,
                                                      PasswordState* pState,
                                                      const char* pURL,
                                                      const char* pOptions,
                                                      const char* pPassword) {
    if (!pOffice || !pState) return NULL;

    pState->pPassword = (char*)pPassword;
    pState->nRequests = 0;

    LoKitDocument* pDoc =
        pOffice->pClass->documentLoadWithOptions(pOffice, pURL, pOptions);

    pState->pPassword = NULL;

    return pDoc;
}

void lok_bridge_document_destroy(LoKitDocument* pDoc) {
    if (!pDoc) return;
    pDoc->pClass->destroy(pDoc);
}

// LOK saveAs returns 0 on failure, unlike typical C convention.
int lok_bridge_document_save_as(LoKitDocument* pDoc, const char* pURL,
                                const char* pFormat,
                                const char* pFilterOptions) {
    if (!pDoc) return 0;
    return pDoc->pClass->saveAs(pDoc, pURL, pFormat, pFilterOptions);
}

int lok_bridge_document_get_type(LoKitDocument* pDoc) {
    if (!pDoc) return -1;
    return pDoc->pClass->getDocumentType(pDoc);
}
