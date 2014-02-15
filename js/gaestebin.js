angular.module('gaestebin', ['ngResource'])
    .config(function($locationProvider) {
        // Enable HTML5 mode
        $locationProvider.html5Mode(true);
    })
    .factory('Paste', function($resource) {
        // Resource for the v1 Paste API
        var Paste = $resource('/paste/v1/:pasteId', {}, {
            create: {method: 'POST', url: '/paste/v1/'}
        });
        return Paste;
    })
    .controller('PasteCtrl', function($scope, $sce, $location, Paste) {
        // Helper to create a Paste and highlight the contents
        var highlightPaste = function(data) {
            $scope.paste = data;
            var highlighted = hljs.highlightAuto(data.Content);
            $scope.paste.highlighted = $sce.trustAsHtml(highlighted.value);
        }

        $scope.resetPaste = function() {
            $scope.paste = undefined;
            $scope.pasteContent = undefined;
            $scope.pasteTitle = undefined;
        };

        // Called when users submit a new paste
        $scope.newPaste = function() {
            // Create new Paste which we'll send as POST body
            var newPaste = new Paste();
            newPaste.Content = $scope.pasteContent;
            newPaste.Title = $scope.pasteTitle;
            var highlighted = hljs.highlightAuto($scope.pasteContent);
            newPaste.Language = highlighted.language;
            // Send POST
            Paste.create(newPaste, function(data) {
                // Reset scope.paste and update URL in browser
                // Does *not* reload the page
                highlightPaste(data);
                $location.path('/' + data.Id);
            });
        };

        $scope.deletePaste = function() {
            console.log($scope.paste);
            Paste.delete({pasteId: $scope.paste.Id}, function(data) {
                console.log("Delete completed")
                console.log(data)
                $scope.resetPaste();
            });
        };

        // Used for displaying the Paste URL
        $scope.baseUrl = $location.absUrl().replace($location.path(), "");
        // Extract pasteId from URL if present
        var pasteId = $location.path().substring(1);
        // Only issue a GET if:
        // - we have a pasteID and
        // - scope.paste has an id that doesn't match
        // This allows us to display new pastes without a page reload
        if (pasteId.length > 0 && (!$scope.paste || $scope.paste.Id != pasteId)) {
            Paste.get({pasteId: pasteId}, function(data) {
                highlightPaste(data);
            }, function(response) {
                console.log(response);
            });
        }
    });
